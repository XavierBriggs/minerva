"""
Odds API Ingestion

Fetches NBA betting odds (team odds and player props) from The Odds API.

API Documentation: https://the-odds-api.com/liveapi/guides/v4/
"""

from datetime import datetime, timedelta
from typing import Dict, List, Optional, Tuple
import requests
from sqlalchemy.orm import Session

from config import config
from Gringotts.store.database import DatabaseManager
from Gringotts.store.models import Game, Team, Player, TeamOdds, PlayerPropOdds, OddsAPIMapping
from src.utils.logger import logger


class OddsAPIIngestion:
    """Ingest betting odds from The Odds API"""
    
    BASE_URL = config.ODDS_API_BASE_URL
    SPORT = "basketball_nba"
    
    # Markets
    MARKETS = {
        "h2h": "h2h",  # Moneyline
        "spreads": "spreads",  # Point spreads
        "totals": "totals",  # Over/Under
    }
    
    # Player props markets
    PLAYER_PROPS_MARKETS = {
        "player_points": "player_points",
        "player_rebounds": "player_rebounds",
        "player_assists": "player_assists",
        "player_threes": "player_threes",
        "player_steals": "player_steals",
        "player_blocks": "player_blocks",
        "player_turnovers": "player_turnovers",
        "player_points_rebounds_assists": "player_points_rebounds_assists",
        "player_points_rebounds": "player_points_rebounds",
        "player_points_assists": "player_points_assists",
        "player_rebounds_assists": "player_rebounds_assists",
    }
    
    def __init__(self):
        """Initialize Odds API ingestion"""
        self.db = DatabaseManager()
        self.session = requests.Session()
        self.api_key = config.ODDS_API_KEY
        
        if not self.api_key:
            logger.warning("⚠️  No Odds API key configured. Set ODDS_API_KEY in .env file")
    
    def fetch_game_odds(
        self,
        markets: Optional[List[str]] = None,
        regions: str = "us",
        odds_format: str = "american"
    ) -> List[Dict]:
        """
        Fetch game odds from The Odds API
        
        Args:
            markets: List of markets to fetch (h2h, spreads, totals)
            regions: Regions to fetch odds from (us, uk, eu, au)
            odds_format: Odds format (american, decimal, fractional)
        
        Returns:
            List of game odds dictionaries
        """
        if not self.api_key:
            logger.error("Cannot fetch odds without API key")
            return []
        
        if markets is None:
            markets = ["h2h", "spreads", "totals"]
        
        url = f"{self.BASE_URL}/sports/{self.SPORT}/odds"
        
        params = {
            "apiKey": self.api_key,
            "regions": regions,
            "markets": ",".join(markets),
            "oddsFormat": odds_format,
        }
        
        logger.info(f"Fetching NBA game odds from The Odds API...")
        
        try:
            response = self.session.get(url, params=params, timeout=15)
            response.raise_for_status()
            
            data = response.json()
            
            logger.info(f"✓ Fetched odds for {len(data)} games")
            
            # Log remaining requests
            remaining = response.headers.get("x-requests-remaining")
            if remaining:
                logger.info(f"API requests remaining: {remaining}")
            
            return data
        
        except requests.exceptions.RequestException as e:
            logger.error(f"Error fetching game odds: {e}")
            return []
    
    def fetch_player_props(
        self,
        markets: Optional[List[str]] = None,
        regions: str = "us",
        odds_format: str = "american"
    ) -> List[Dict]:
        """
        Fetch player prop odds from The Odds API
        
        Args:
            markets: List of player prop markets to fetch
            regions: Regions to fetch odds from
            odds_format: Odds format
        
        Returns:
            List of player props dictionaries
        """
        if not self.api_key:
            logger.error("Cannot fetch player props without API key")
            return []
        
        if markets is None:
            markets = list(self.PLAYER_PROPS_MARKETS.keys())
        
        all_props = []
        
        for market in markets:
            url = f"{self.BASE_URL}/sports/{self.SPORT}/events"
            
            params = {
                "apiKey": self.api_key,
                "regions": regions,
                "markets": market,
                "oddsFormat": odds_format,
            }
            
            logger.info(f"Fetching player props for market: {market}")
            
            try:
                response = self.session.get(url, params=params, timeout=15)
                response.raise_for_status()
                
                data = response.json()
                
                # Add market type to each event
                for event in data:
                    event['market_type'] = market
                
                all_props.extend(data)
                
                logger.info(f"✓ Fetched props for {len(data)} events (market: {market})")
                
            except requests.exceptions.RequestException as e:
                logger.error(f"Error fetching player props for {market}: {e}")
                continue
        
        return all_props
    
    def ingest_game_odds(self, date: Optional[datetime] = None) -> Tuple[int, int]:
        """
        Ingest game odds and store in database
        
        Args:
            date: Date to fetch odds for (default: today/upcoming)
        
        Returns:
            Tuple of (games_updated, odds_records_created)
        """
        odds_data = self.fetch_game_odds()
        
        if not odds_data:
            logger.warning("No odds data to ingest")
            return 0, 0
        
        games_updated = 0
        odds_created = 0
        
        with self.db.get_session() as session:
            for event in odds_data:
                try:
                    success, num_odds = self._ingest_event_odds(event, session)
                    if success:
                        games_updated += 1
                        odds_created += num_odds
                except Exception as e:
                    logger.error(f"Error ingesting odds for event: {e}")
                    continue
            
            session.commit()
        
        logger.info(f"✅ Ingested odds for {games_updated} games ({odds_created} odds records)")
        return games_updated, odds_created
    
    def _ingest_event_odds(self, event: Dict, session: Session) -> Tuple[bool, int]:
        """
        Ingest odds for a single event
        
        Args:
            event: Event data from Odds API
            session: Database session
        
        Returns:
            Tuple of (success, num_odds_created)
        """
        odds_api_event_id = event.get("id")
        home_team_name = event.get("home_team")
        away_team_name = event.get("away_team")
        commence_time = event.get("commence_time")
        
        if not all([odds_api_event_id, home_team_name, away_team_name]):
            logger.warning("Missing required event data")
            return False, 0
        
        # Find corresponding game in database
        game = self._find_game_by_teams_and_date(
            home_team_name, away_team_name, commence_time, session
        )
        
        if not game:
            logger.debug(f"Game not found for: {away_team_name} @ {home_team_name}")
            return False, 0
        
        # Store mapping
        self._create_odds_mapping(game.game_id, odds_api_event_id, session)
        
        # Ingest bookmaker odds
        bookmakers = event.get("bookmakers", [])
        odds_count = 0
        
        for bookmaker in bookmakers:
            bookmaker_name = bookmaker.get("key")
            markets = bookmaker.get("markets", [])
            
            for market in markets:
                market_key = market.get("key")
                outcomes = market.get("outcomes", [])
                
                odds_count += self._ingest_market_odds(
                    game=game,
                    bookmaker_name=bookmaker_name,
                    market_key=market_key,
                    outcomes=outcomes,
                    home_team_name=home_team_name,
                    away_team_name=away_team_name,
                    session=session
                )
        
        return True, odds_count
    
    def _ingest_market_odds(
        self,
        game: Game,
        bookmaker_name: str,
        market_key: str,
        outcomes: List[Dict],
        home_team_name: str,
        away_team_name: str,
        session: Session
    ) -> int:
        """Ingest odds for a specific market"""
        odds_count = 0
        
        for outcome in outcomes:
            team_name = outcome.get("name")
            price = outcome.get("price")  # American odds
            point = outcome.get("point")  # For spreads and totals
            
            # Determine which team
            if team_name == home_team_name:
                team_id = game.home_team_id
            elif team_name == away_team_name:
                team_id = game.away_team_id
            elif team_name in ["Over", "Under"]:
                # Totals don't have team-specific odds
                team_id = game.home_team_id  # Store under home team
            else:
                continue
            
            # Check if odds record exists
            existing = session.query(TeamOdds).filter_by(
                game_id=game.game_id,
                team_id=team_id,
                bookmaker=bookmaker_name
            ).first()
            
            if market_key == "h2h":
                # Moneyline
                if existing:
                    existing.moneyline = price
                else:
                    new_odds = TeamOdds(
                        game_id=game.game_id,
                        team_id=team_id,
                        bookmaker=bookmaker_name,
                        moneyline=price
                    )
                    session.add(new_odds)
                odds_count += 1
            
            elif market_key == "spreads":
                # Point spread
                if existing:
                    existing.spread_points = point
                    existing.spread_odds = price
                else:
                    new_odds = TeamOdds(
                        game_id=game.game_id,
                        team_id=team_id,
                        bookmaker=bookmaker_name,
                        spread_points=point,
                        spread_odds=price
                    )
                    session.add(new_odds)
                odds_count += 1
            
            elif market_key == "totals":
                # Over/Under
                if team_name == "Over":
                    if existing:
                        existing.total_points = point
                        existing.over_odds = price
                    else:
                        new_odds = TeamOdds(
                            game_id=game.game_id,
                            team_id=team_id,
                            bookmaker=bookmaker_name,
                            total_points=point,
                            over_odds=price
                        )
                        session.add(new_odds)
                    odds_count += 1
                
                elif team_name == "Under":
                    if existing:
                        existing.under_odds = price
                    else:
                        # Find the Over record we just created
                        over_record = session.query(TeamOdds).filter_by(
                            game_id=game.game_id,
                            team_id=team_id,
                            bookmaker=bookmaker_name
                        ).first()
                        if over_record:
                            over_record.under_odds = price
        
        return odds_count
    
    def _find_game_by_teams_and_date(
        self,
        home_team_name: str,
        away_team_name: str,
        commence_time_str: str,
        session: Session
    ) -> Optional[Game]:
        """
        Find game in database by team names and date
        
        Args:
            home_team_name: Home team name from Odds API
            away_team_name: Away team name from Odds API
            commence_time_str: ISO format datetime string
            session: Database session
        
        Returns:
            Game object or None
        """
        # Parse commence time
        try:
            commence_time = datetime.fromisoformat(commence_time_str.replace("Z", "+00:00"))
            game_date = commence_time.date()
        except:
            logger.warning(f"Could not parse commence time: {commence_time_str}")
            return None
        
        # Map Odds API team names to database team abbreviations
        home_team = self._map_team_name(home_team_name, session)
        away_team = self._map_team_name(away_team_name, session)
        
        if not home_team or not away_team:
            return None
        
        # Find game
        game = session.query(Game).filter_by(
            game_date=game_date,
            home_team_id=home_team.team_id,
            away_team_id=away_team.team_id
        ).first()
        
        return game
    
    def _map_team_name(self, team_name: str, session: Session) -> Optional[Team]:
        """
        Map Odds API team name to database team
        
        Args:
            team_name: Team name from Odds API
            session: Database session
        
        Returns:
            Team object or None
        """
        # Common mapping patterns
        team_name_map = {
            "Atlanta Hawks": "ATL",
            "Boston Celtics": "BOS",
            "Brooklyn Nets": "BKN",
            "Charlotte Hornets": "CHA",
            "Chicago Bulls": "CHI",
            "Cleveland Cavaliers": "CLE",
            "Dallas Mavericks": "DAL",
            "Denver Nuggets": "DEN",
            "Detroit Pistons": "DET",
            "Golden State Warriors": "GS",
            "Houston Rockets": "HOU",
            "Indiana Pacers": "IND",
            "Los Angeles Clippers": "LAC",
            "Los Angeles Lakers": "LAL",
            "Memphis Grizzlies": "MEM",
            "Miami Heat": "MIA",
            "Milwaukee Bucks": "MIL",
            "Minnesota Timberwolves": "MIN",
            "New Orleans Pelicans": "NO",
            "New York Knicks": "NY",
            "Oklahoma City Thunder": "OKC",
            "Orlando Magic": "ORL",
            "Philadelphia 76ers": "PHI",
            "Phoenix Suns": "PHX",
            "Portland Trail Blazers": "POR",
            "Sacramento Kings": "SAC",
            "San Antonio Spurs": "SA",
            "Toronto Raptors": "TOR",
            "Utah Jazz": "UTAH",
            "Washington Wizards": "WSH",
        }
        
        abbreviation = team_name_map.get(team_name)
        if not abbreviation:
            logger.warning(f"Unknown team name from Odds API: {team_name}")
            return None
        
        team = session.query(Team).filter_by(team_abbreviation=abbreviation).first()
        return team
    
    def _create_odds_mapping(
        self,
        espn_game_id: str,
        odds_api_event_id: str,
        session: Session
    ):
        """Create mapping between ESPN game ID and Odds API event ID"""
        existing = session.query(OddsAPIMapping).filter_by(
            espn_game_id=espn_game_id
        ).first()
        
        if not existing:
            mapping = OddsAPIMapping(
                espn_game_id=espn_game_id,
                odds_api_event_id=odds_api_event_id
            )
            session.add(mapping)
    
    def get_api_usage(self) -> Optional[Dict]:
        """
        Get API usage statistics
        
        Returns:
            Dictionary with usage info or None
        """
        if not self.api_key:
            return None
        
        url = f"{self.BASE_URL}/sports/{self.SPORT}/odds"
        params = {"apiKey": self.api_key}
        
        try:
            response = self.session.get(url, params=params, timeout=10)
            
            return {
                "requests_remaining": response.headers.get("x-requests-remaining"),
                "requests_used": response.headers.get("x-requests-used"),
            }
        except:
            return None

