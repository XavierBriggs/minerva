"""
ESPN NBA Stats Ingestion

Fetches NBA game and player statistics from ESPN's API endpoints.
"""

import re
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Tuple

import pytz
import requests
from sqlalchemy.orm import Session

from Gringotts.ingest.player_mapping import PlayerIDMapper
from Gringotts.store.database import DatabaseManager
from Gringotts.store.models import Game, PlayerGameStats, Team
from src.utils.logger import logger


class ESPNStatsIngestion:
    """Ingest NBA statistics from ESPN API"""
    
    BASE_URL = "https://site.api.espn.com/apis/site/v2/sports/basketball/nba"
    
    # ESPN stat indices in the stats array
    # Based on: MIN, PTS, OREB, DREB, REB, AST, STL, BLK, TO, FG, FG%, 3PT, 3PT%, FT, FT%, PF, +/-
    STAT_INDICES = {
        "minutes": 0,
        "points": 1,
        "offensive_rebounds": 2,
        "defensive_rebounds": 3,
        "rebounds": 4,
        "assists": 5,
        "steals": 6,
        "blocks": 7,
        "turnovers": 8,
        "field_goals": 9,  # "X-Y" format
        "field_goal_pct": 10,
        "three_pointers": 11,  # "X-Y" format
        "three_point_pct": 12,
        "free_throws": 13,  # "X-Y" format
        "free_throw_pct": 14,
        "personal_fouls": 15,
        "plus_minus": 16,
    }
    
    def __init__(self):
        """Initialize ESPN stats ingestion"""
        self.db = DatabaseManager()
        self.player_mapper = PlayerIDMapper()
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'
        })
    
    def fetch_scoreboard(self, date: Optional[datetime] = None) -> Dict:
        """
        Fetch scoreboard for a specific date
        
        Args:
            date: Date to fetch games for (defaults to yesterday)
        
        Returns:
            Scoreboard data dictionary
        """
        if date is None:
            # Default to yesterday (games completed by 12pm EST today)
            date = datetime.now() - timedelta(days=1)
        
        date_str = date.strftime("%Y%m%d")
        url = f"{self.BASE_URL}/scoreboard"
        params = {"dates": date_str}
        
        logger.info(f"Fetching ESPN scoreboard for {date.date()}")
        
        try:
            response = self.session.get(url, params=params, timeout=15)
            response.raise_for_status()
            data = response.json()
            
            events = data.get("events", [])
            logger.info(f"✓ Fetched {len(events)} games from ESPN scoreboard")
            
            return data
        
        except requests.exceptions.RequestException as e:
            logger.error(f"Error fetching ESPN scoreboard: {e}")
            return {}
    
    def fetch_game_summary(self, game_id: str) -> Dict:
        """
        Fetch detailed game summary with box scores
        
        Args:
            game_id: ESPN game ID
        
        Returns:
            Game summary data dictionary
        """
        url = f"{self.BASE_URL}/summary"
        params = {"event": game_id}
        
        logger.debug(f"Fetching ESPN game summary for {game_id}")
        
        try:
            response = self.session.get(url, params=params, timeout=15)
            response.raise_for_status()
            data = response.json()
            
            logger.debug(f"✓ Fetched game summary for {game_id}")
            return data
        
        except requests.exceptions.RequestException as e:
            logger.error(f"Error fetching ESPN game summary for {game_id}: {e}")
            return {}
    
    def ingest_games_for_date(self, date: Optional[datetime] = None) -> Tuple[int, int]:
        """
        Ingest all games and player stats for a specific date
        
        Args:
            date: Date to ingest (defaults to yesterday)
        
        Returns:
            Tuple of (games_ingested, player_stats_ingested)
        """
        scoreboard = self.fetch_scoreboard(date)
        if not scoreboard:
            logger.warning("No scoreboard data fetched")
            return 0, 0
        
        events = scoreboard.get("events", [])
        if not events:
            logger.info(f"No games found for {date.date() if date else 'yesterday'}")
            return 0, 0
        
        games_count = 0
        stats_count = 0
        
        with self.db.get_session() as session:
            # Load player mappings
            self.player_mapper.load_cache_from_db(session)
            
            for event in events:
                # Only ingest completed games
                status = event.get("status", {})
                is_completed = status.get("type", {}).get("completed", False)
                
                if not is_completed:
                    game_id = event.get("id", "unknown")
                    logger.info(f"Skipping incomplete game {game_id}")
                    continue
                
                # Ingest game and stats
                success, num_stats = self.ingest_game(event, session)
                if success:
                    games_count += 1
                    stats_count += num_stats
        
        logger.info(f"✅ Ingested {games_count} games with {stats_count} player stat records")
        return games_count, stats_count
    
    def ingest_game(self, event: Dict, session: Session) -> Tuple[bool, int]:
        """
        Ingest a single game and its player stats
        
        Args:
            event: Game event data from scoreboard
            session: Database session
        
        Returns:
            Tuple of (success, num_player_stats)
        """
        try:
            game_id = event.get("id")
            if not game_id:
                logger.warning("Event missing game ID")
                return False, 0
            
            # Parse game basic info from scoreboard
            # ESPN returns UTC time, convert to US Eastern to get correct game date
            game_date_str = event.get("date")
            game_date = self._parse_game_date(game_date_str) if game_date_str else None
            
            # Get team and score info from competitions
            competitions = event.get("competitions", [])
            if not competitions:
                logger.warning(f"No competition data for game {game_id}")
                return False, 0
            
            comp = competitions[0]
            competitors = comp.get("competitors", [])
            if len(competitors) < 2:
                logger.warning(f"Insufficient competitor data for game {game_id}")
                return False, 0
            
            # ESPN typically has index 0 = home, index 1 = away
            home_comp = next((c for c in competitors if c.get("homeAway") == "home"), None)
            away_comp = next((c for c in competitors if c.get("homeAway") == "away"), None)

            if not home_comp or not away_comp:
                logger.warning(f"Could not identify home/away teams for game {game_id}")
                return False, 0

            # Get team abbreviations directly from ESPN API (more reliable than ESPN IDs)
            home_team_abbr = home_comp.get("team", {}).get("abbreviation")
            away_team_abbr = away_comp.get("team", {}).get("abbreviation")
            home_score = int(home_comp.get("score", 0))
            away_score = int(away_comp.get("score", 0))

            # Map team abbreviations to our database team IDs
            home_team = self._get_team_by_abbreviation(home_team_abbr, session)
            away_team = self._get_team_by_abbreviation(away_team_abbr, session)
            
            if not home_team or not away_team:
                logger.warning(f"Could not map teams for game {game_id}")
                return False, 0
            
            # Determine season_id from date (e.g., 2025-26 season)
            # Season spans Oct-Jun, so Oct 2025 game = "2025-26" season
            season_id = self._get_season_id(game_date)
            
            # Create or update game record
            game = session.query(Game).filter_by(game_id=game_id).first()
            
            if not game:
                game = Game(
                    game_id=game_id,
                    season_id=season_id,
                    game_date=game_date,
                    home_team_id=home_team.team_id,
                    away_team_id=away_team.team_id,
                    home_score=home_score,
                    away_score=away_score,
                    game_status="Final",
                    season_type="Regular Season",  # Could parse from event if needed
                )
                session.add(game)
                logger.info(f"Created game record: {game_id}")
            else:
                # Update scores and status
                game.home_score = home_score
                game.away_score = away_score
                game.game_status = "Final"
                logger.info(f"Updated game record: {game_id}")
            
            session.flush()  # Flush to get game in database for foreign keys
            
            # Fetch detailed box scores
            summary = self.fetch_game_summary(game_id)
            if not summary:
                logger.warning(f"No summary data for game {game_id}")
                session.commit()
                return True, 0  # Game created but no stats
            
            # Ingest player stats
            num_stats = self._ingest_player_stats(
                game_id=game_id,
                summary=summary,
                home_team_id=home_team.team_id,
                away_team_id=away_team.team_id,
                session=session
            )
            
            session.commit()
            logger.info(f"✓ Ingested game {game_id} with {num_stats} player stats")
            
            return True, num_stats
        
        except Exception as e:
            logger.error(f"Error ingesting game: {e}", exc_info=True)
            session.rollback()
            return False, 0
    
    def _ingest_player_stats(
        self,
        game_id: str,
        summary: Dict,
        home_team_id: int,
        away_team_id: int,
        session: Session
    ) -> int:
        """
        Ingest player statistics from game summary
        
        Args:
            game_id: Game ID
            summary: Game summary data
            home_team_id: Home team database ID
            away_team_id: Away team database ID
            session: Database session
        
        Returns:
            Number of player stats records created/updated
        """
        boxscore = summary.get("boxscore", {})
        players_data = boxscore.get("players", [])
        
        if not players_data:
            logger.warning(f"No player data in boxscore for game {game_id}")
            return 0
        
        stats_count = 0
        
        for team_data in players_data:
            team_info = team_data.get("team", {})
            team_abbr = team_info.get("abbreviation")
            
            # Determine which team ID to use
            team_id = home_team_id if team_abbr else away_team_id
            
            # Get player statistics
            statistics = team_data.get("statistics", [])
            if not statistics:
                continue
            
            stat_group = statistics[0]  # First group has individual player stats
            athletes = stat_group.get("athletes", [])
            
            for athlete_data in athletes:
                try:
                    player_stat = self._parse_player_stat(
                        athlete_data=athlete_data,
                        game_id=game_id,
                        team_id=team_id,
                        team_abbr=team_abbr,
                        session=session
                    )
                    
                    if player_stat:
                        session.add(player_stat)
                        stats_count += 1
                
                except Exception as e:
                    logger.error(f"Error parsing player stat: {e}")
                    continue
        
        return stats_count
    
    def _parse_player_stat(
        self,
        athlete_data: Dict,
        game_id: str,
        team_id: int,
        team_abbr: str,
        session: Session
    ) -> Optional[PlayerGameStats]:
        """
        Parse individual player stats from ESPN data
        
        Args:
            athlete_data: Athlete data from boxscore
            game_id: Game ID
            team_id: Team database ID
            team_abbr: Team abbreviation
            session: Database session
        
        Returns:
            PlayerGameStats object or None
        """
        athlete = athlete_data.get("athlete", {})
        espn_player_id = str(athlete.get("id", ""))
        player_name = athlete.get("displayName", "")
        
        if not espn_player_id:
            return None
        
        # Check if player didn't play
        did_not_play = athlete_data.get("didNotPlay", False)
        if did_not_play:
            logger.debug(f"Player {player_name} did not play")
            return None
        
        # Map ESPN player ID to database player ID
        db_player_id = self.player_mapper.map_espn_id_to_player(
            espn_id=espn_player_id,
            player_name=player_name,
            team_abbreviation=team_abbr,
            session=session
        )
        
        if not db_player_id:
            logger.warning(f"Could not map ESPN player {player_name} (ESPN ID: {espn_player_id})")
            return None
        
        # Parse stats array
        stats = athlete_data.get("stats", [])
        if len(stats) < 17:  # Need all 17 stats
            logger.warning(f"Incomplete stats for player {player_name}")
            return None
        
        # Parse shooting stats (format: "X-Y")
        fg_made, fg_att = self._parse_made_attempted(stats[self.STAT_INDICES["field_goals"]])
        three_made, three_att = self._parse_made_attempted(stats[self.STAT_INDICES["three_pointers"]])
        ft_made, ft_att = self._parse_made_attempted(stats[self.STAT_INDICES["free_throws"]])
        
        # Parse minutes (format: "33" or "33:15")
        minutes = self._parse_minutes(stats[self.STAT_INDICES["minutes"]])
        
        # Check if existing stat record
        existing = (
            session.query(PlayerGameStats)
            .filter_by(game_id=game_id, player_id=db_player_id)
            .first()
        )
        
        stat_data = {
            "game_id": game_id,
            "player_id": db_player_id,
            "team_id": team_id,
            "points": int(stats[self.STAT_INDICES["points"]] or 0),
            "rebounds": int(stats[self.STAT_INDICES["rebounds"]] or 0),
            "assists": int(stats[self.STAT_INDICES["assists"]] or 0),
            "steals": int(stats[self.STAT_INDICES["steals"]] or 0),
            "blocks": int(stats[self.STAT_INDICES["blocks"]] or 0),
            "turnovers": int(stats[self.STAT_INDICES["turnovers"]] or 0),
            "field_goals_made": fg_made,
            "field_goals_attempted": fg_att,
            "three_pointers_made": three_made,
            "three_pointers_attempted": three_att,
            "free_throws_made": ft_made,
            "free_throws_attempted": ft_att,
            "minutes_played": minutes,
            "plus_minus": self._parse_plus_minus(stats[self.STAT_INDICES["plus_minus"]]),
            "offensive_rebounds": int(stats[self.STAT_INDICES["offensive_rebounds"]] or 0),
            "defensive_rebounds": int(stats[self.STAT_INDICES["defensive_rebounds"]] or 0),
            "personal_fouls": int(stats[self.STAT_INDICES["personal_fouls"]] or 0),
        }
        
        # Calculate PRA
        stat_data["points_rebounds_assists"] = (
            stat_data["points"] + stat_data["rebounds"] + stat_data["assists"]
        )
        
        if existing:
            # Update existing record
            for key, value in stat_data.items():
                setattr(existing, key, value)
            return existing
        else:
            # Create new record
            return PlayerGameStats(**stat_data)
    
    def _get_team_by_abbreviation(self, abbreviation: str, session: Session) -> Optional[Team]:
        """
        Get team by abbreviation (more reliable than ESPN team ID)

        Args:
            abbreviation: Team abbreviation (e.g., 'ATL', 'IND')
            session: Database session

        Returns:
            Team object or None
        """
        if not abbreviation:
            logger.warning("Team abbreviation is empty")
            return None

        team = session.query(Team).filter_by(team_abbreviation=abbreviation).first()
        if not team:
            logger.warning(f"Team not found in database: {abbreviation}")

        return team
    
    @staticmethod
    def _parse_game_date(game_date_str: str):
        """
        Parse ESPN game date from UTC to US Eastern time to get correct date

        ESPN returns dates in UTC (e.g., "2025-10-31T00:00Z"), but games are
        played in US timezones. Converting to Eastern time ensures we get the
        correct calendar date for the game.

        Args:
            game_date_str: ISO format date string from ESPN (UTC)

        Returns:
            Date object in US Eastern timezone
        """
        if not game_date_str:
            return None

        try:
            # Parse UTC datetime
            dt_utc = datetime.fromisoformat(game_date_str.replace("Z", "+00:00"))

            # Convert to US Eastern timezone
            eastern = pytz.timezone('US/Eastern')
            dt_eastern = dt_utc.astimezone(eastern)

            # Return the date in Eastern time
            return dt_eastern.date()
        except Exception as e:
            logger.error(f"Error parsing game date '{game_date_str}': {e}")
            return None

    @staticmethod
    def _parse_made_attempted(stat_str: str) -> Tuple[int, int]:
        """Parse 'X-Y' format into made and attempted"""
        if not stat_str or stat_str == "0":
            return 0, 0
        
        try:
            parts = stat_str.split("-")
            if len(parts) == 2:
                made = int(parts[0])
                attempted = int(parts[1])
                return made, attempted
        except (ValueError, IndexError):
            pass
        
        return 0, 0
    
    @staticmethod
    def _parse_minutes(minutes_str: str) -> float:
        """Parse minutes string into float"""
        if not minutes_str or minutes_str == "0":
            return 0.0
        
        try:
            # Handle "33" or "33:15" format
            if ":" in minutes_str:
                parts = minutes_str.split(":")
                mins = int(parts[0])
                secs = int(parts[1]) if len(parts) > 1 else 0
                return mins + (secs / 60.0)
            else:
                return float(minutes_str)
        except (ValueError, IndexError):
            return 0.0
    
    @staticmethod
    def _parse_plus_minus(pm_str: str) -> Optional[int]:
        """Parse plus/minus string"""
        if not pm_str or pm_str == "0":
            return 0
        
        try:
            # Remove '+' prefix if present
            pm_str = pm_str.replace("+", "")
            return int(pm_str)
        except ValueError:
            return None
    
    @staticmethod
    def _get_season_id(game_date: datetime) -> str:
        """
        Determine season ID from game date
        NBA season runs from October to June.
        Season ID format: "2025-26"
        
        Examples:
            - Game on Oct 15, 2025 → "2025-26"
            - Game on Jan 10, 2026 → "2025-26"
            - Game on Oct 20, 2026 → "2026-27"
        
        Args:
            game_date: Date of the game
        
        Returns:
            Season ID string
        """
        year = game_date.year
        month = game_date.month
        
        # If game is Oct-Dec, it's the start year
        # If game is Jan-Jun, it's the end year (subtract 1 for start)
        if month >= 10:
            start_year = year
        else:
            start_year = year - 1
        
        end_year = start_year + 1
        return f"{start_year}-{str(end_year)[-2:]}"

