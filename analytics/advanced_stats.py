"""
Advanced NBA Statistics Calculator

Calculates advanced basketball metrics from basic box score statistics.
All formulas are based on industry-standard basketball analytics.

References:
- Basketball Reference (basketball-reference.com)
- Dean Oliver's "Basketball on Paper"
- John Hollinger's PER formula
"""

from typing import Dict, Optional, Tuple
from dataclasses import dataclass
import math


@dataclass
class PlayerBoxScore:
    """Container for player box score statistics"""
    # Basic stats
    minutes_played: float
    points: int
    field_goals_made: int
    field_goals_attempted: int
    three_pointers_made: int
    three_pointers_attempted: int
    free_throws_made: int
    free_throws_attempted: int
    offensive_rebounds: int
    defensive_rebounds: int
    rebounds: int
    assists: int
    steals: int
    blocks: int
    turnovers: int
    personal_fouls: int
    
    # Team context (needed for percentage stats)
    team_minutes: float = 240.0  # 48 minutes * 5 players
    team_field_goals_made: Optional[int] = None
    team_field_goals_attempted: Optional[int] = None
    team_three_pointers_attempted: Optional[int] = None
    team_free_throws_attempted: Optional[int] = None
    team_offensive_rebounds: Optional[int] = None
    team_defensive_rebounds: Optional[int] = None
    team_rebounds: Optional[int] = None
    team_assists: Optional[int] = None
    team_turnovers: Optional[int] = None
    
    # Opponent stats (for percentage calculations)
    opponent_field_goals_attempted: Optional[int] = None
    opponent_three_pointers_attempted: Optional[int] = None
    opponent_offensive_rebounds: Optional[int] = None
    opponent_defensive_rebounds: Optional[int] = None
    opponent_rebounds: Optional[int] = None


@dataclass
class TeamBoxScore:
    """Container for team box score statistics"""
    minutes_played: float = 240.0  # 48 minutes * 5 players
    points: int = 0
    field_goals_made: int = 0
    field_goals_attempted: int = 0
    three_pointers_made: int = 0
    three_pointers_attempted: int = 0
    free_throws_made: int = 0
    free_throws_attempted: int = 0
    offensive_rebounds: int = 0
    defensive_rebounds: int = 0
    rebounds: int = 0
    assists: int = 0
    steals: int = 0
    blocks: int = 0
    turnovers: int = 0
    personal_fouls: int = 0


class AdvancedStatsCalculator:
    """Calculate advanced NBA statistics"""
    
    # Constants
    LEAGUE_AVERAGE_PTS_PER_POSSESSION = 1.07  # Approximate, adjust per season
    
    @staticmethod
    def true_shooting_percentage(points: int, fga: int, fta: int) -> Optional[float]:
        """
        Calculate True Shooting Percentage (TS%)
        
        Measures a player's shooting efficiency by accounting for 2PT, 3PT, and FT.
        
        Formula: TS% = PTS / (2 * (FGA + 0.44 * FTA))
        
        The 0.44 coefficient estimates the proportion of FTA that represent possessions.
        
        Args:
            points: Total points scored
            fga: Field goal attempts
            fta: Free throw attempts
        
        Returns:
            TS% as a decimal (e.g., 0.550 for 55.0%), or None if undefined
        """
        denominator = 2 * (fga + 0.44 * fta)
        if denominator == 0:
            return None
        return points / denominator
    
    @staticmethod
    def effective_field_goal_percentage(fgm: int, three_pm: int, fga: int) -> Optional[float]:
        """
        Calculate Effective Field Goal Percentage (eFG%)
        
        Adjusts FG% to account for the fact that 3PT shots are worth more than 2PT shots.
        
        Formula: eFG% = (FGM + 0.5 * 3PM) / FGA
        
        Args:
            fgm: Field goals made
            three_pm: Three-pointers made
            fga: Field goal attempts
        
        Returns:
            eFG% as a decimal (e.g., 0.520 for 52.0%), or None if undefined
        """
        if fga == 0:
            return None
        return (fgm + 0.5 * three_pm) / fga
    
    @staticmethod
    def usage_percentage(
        fga: int,
        fta: int,
        tov: int,
        mp: float,
        team_fga: int,
        team_fta: int,
        team_tov: int,
        team_mp: float = 240.0
    ) -> Optional[float]:
        """
        Calculate Usage Percentage (USG%)
        
        Estimates the percentage of team plays used by a player while on the court.
        
        Formula: USG% = 100 * ((FGA + 0.44 * FTA + TOV) * (TmMP / 5)) / 
                        (MP * (TmFGA + 0.44 * TmFTA + TmTOV))
        
        Args:
            fga: Player field goal attempts
            fta: Player free throw attempts
            tov: Player turnovers
            mp: Player minutes played
            team_fga: Team field goal attempts
            team_fta: Team free throw attempts
            team_tov: Team turnovers
            team_mp: Team total minutes (default 240)
        
        Returns:
            USG% as a percentage (e.g., 25.5), or None if undefined
        """
        if mp == 0:
            return None
        
        player_possessions = fga + 0.44 * fta + tov
        team_possessions = team_fga + 0.44 * team_fta + team_tov
        
        if team_possessions == 0:
            return None
        
        return 100 * (player_possessions * (team_mp / 5)) / (mp * team_possessions)
    
    @staticmethod
    def assist_percentage(
        ast: int,
        mp: float,
        team_fgm: int,
        team_mp: float,
        player_fgm: int
    ) -> Optional[float]:
        """
        Calculate Assist Percentage (AST%)
        
        Estimates the percentage of teammate field goals a player assisted while on court.
        
        Formula: AST% = 100 * AST / (((MP / (TmMP / 5)) * TmFG) - FG)
        
        Args:
            ast: Player assists
            mp: Player minutes played
            team_fgm: Team field goals made
            team_mp: Team total minutes
            player_fgm: Player's own field goals made
        
        Returns:
            AST% as a percentage (e.g., 28.5), or None if undefined
        """
        if mp == 0 or team_mp == 0:
            return None
        
        teammate_fgm = ((mp / (team_mp / 5)) * team_fgm) - player_fgm
        
        if teammate_fgm <= 0:
            return None
        
        return 100 * ast / teammate_fgm
    
    @staticmethod
    def rebound_percentage(
        reb: int,
        mp: float,
        team_reb: int,
        opponent_reb: int,
        team_mp: float = 240.0
    ) -> Optional[float]:
        """
        Calculate Rebound Percentage (REB%)
        
        Estimates the percentage of available rebounds a player grabbed while on court.
        
        Formula: REB% = 100 * (REB * (TmMP / 5)) / (MP * (TmREB + OppREB))
        
        Args:
            reb: Player rebounds
            mp: Player minutes played
            team_reb: Team total rebounds
            opponent_reb: Opponent total rebounds
            team_mp: Team total minutes
        
        Returns:
            REB% as a percentage (e.g., 12.5), or None if undefined
        """
        if mp == 0:
            return None
        
        available_rebounds = team_reb + opponent_reb
        
        if available_rebounds == 0:
            return None
        
        return 100 * (reb * (team_mp / 5)) / (mp * available_rebounds)
    
    @staticmethod
    def offensive_rebound_percentage(
        oreb: int,
        mp: float,
        team_oreb: int,
        opponent_dreb: int,
        team_mp: float = 240.0
    ) -> Optional[float]:
        """
        Calculate Offensive Rebound Percentage (ORB%)
        
        Estimates the percentage of available offensive rebounds a player grabbed.
        
        Formula: ORB% = 100 * (ORB * (TmMP / 5)) / (MP * (TmORB + OppDREB))
        
        Args:
            oreb: Player offensive rebounds
            mp: Player minutes played
            team_oreb: Team offensive rebounds
            opponent_dreb: Opponent defensive rebounds
            team_mp: Team total minutes
        
        Returns:
            ORB% as a percentage, or None if undefined
        """
        if mp == 0:
            return None
        
        available_oreb = team_oreb + opponent_dreb
        
        if available_oreb == 0:
            return None
        
        return 100 * (oreb * (team_mp / 5)) / (mp * available_oreb)
    
    @staticmethod
    def defensive_rebound_percentage(
        dreb: int,
        mp: float,
        team_dreb: int,
        opponent_oreb: int,
        team_mp: float = 240.0
    ) -> Optional[float]:
        """
        Calculate Defensive Rebound Percentage (DRB%)
        
        Estimates the percentage of available defensive rebounds a player grabbed.
        
        Formula: DRB% = 100 * (DRB * (TmMP / 5)) / (MP * (TmDRB + OppOREB))
        
        Args:
            dreb: Player defensive rebounds
            mp: Player minutes played
            team_dreb: Team defensive rebounds
            opponent_oreb: Opponent offensive rebounds
            team_mp: Team total minutes
        
        Returns:
            DRB% as a percentage, or None if undefined
        """
        if mp == 0:
            return None
        
        available_dreb = team_dreb + opponent_oreb
        
        if available_dreb == 0:
            return None
        
        return 100 * (dreb * (team_mp / 5)) / (mp * available_dreb)
    
    @staticmethod
    def turnover_percentage(fga: int, fta: int, tov: int) -> Optional[float]:
        """
        Calculate Turnover Percentage (TOV%)
        
        Estimates the percentage of a player's possessions that end in a turnover.
        
        Formula: TOV% = 100 * TOV / (FGA + 0.44 * FTA + TOV)
        
        Args:
            fga: Field goal attempts
            fta: Free throw attempts
            tov: Turnovers
        
        Returns:
            TOV% as a percentage (e.g., 12.5), or None if undefined
        """
        possessions = fga + 0.44 * fta + tov
        
        if possessions == 0:
            return None
        
        return 100 * tov / possessions
    
    @staticmethod
    def steal_percentage(
        stl: int,
        mp: float,
        opponent_possessions: int,
        team_mp: float = 240.0
    ) -> Optional[float]:
        """
        Calculate Steal Percentage (STL%)
        
        Estimates the percentage of opponent possessions that end with a steal.
        
        Formula: STL% = 100 * STL / ((MP / (TmMP / 5)) * OppPoss)
        
        Args:
            stl: Player steals
            mp: Player minutes played
            opponent_possessions: Opponent total possessions
            team_mp: Team total minutes
        
        Returns:
            STL% as a percentage, or None if undefined
        """
        if mp == 0 or team_mp == 0:
            return None
        
        player_share = mp / (team_mp / 5)
        player_opponent_possessions = player_share * opponent_possessions
        
        if player_opponent_possessions == 0:
            return None
        
        return 100 * stl / player_opponent_possessions
    
    @staticmethod
    def block_percentage(
        blk: int,
        mp: float,
        opponent_fga: int,
        opponent_three_pa: int,
        team_mp: float = 240.0
    ) -> Optional[float]:
        """
        Calculate Block Percentage (BLK%)
        
        Estimates the percentage of opponent 2PT attempts blocked while on court.
        
        Formula: BLK% = 100 * BLK / ((MP / (TmMP / 5)) * (OppFGA - Opp3PA))
        
        Args:
            blk: Player blocks
            mp: Player minutes played
            opponent_fga: Opponent field goal attempts
            opponent_three_pa: Opponent three-point attempts
            team_mp: Team total minutes
        
        Returns:
            BLK% as a percentage, or None if undefined
        """
        if mp == 0 or team_mp == 0:
            return None
        
        player_share = mp / (team_mp / 5)
        opponent_two_point_attempts = opponent_fga - opponent_three_pa
        player_opponent_2pa = player_share * opponent_two_point_attempts
        
        if player_opponent_2pa == 0:
            return None
        
        return 100 * blk / player_opponent_2pa
    
    @staticmethod
    def free_throw_rate(fta: int, fga: int) -> Optional[float]:
        """
        Calculate Free Throw Rate (FTr)
        
        Measures how often a player gets to the free throw line.
        
        Formula: FTr = FTA / FGA
        
        Args:
            fta: Free throw attempts
            fga: Field goal attempts
        
        Returns:
            FTr as a decimal (e.g., 0.25), or None if undefined
        """
        if fga == 0:
            return None
        return fta / fga
    
    @staticmethod
    def points_per_possession(points: int, fga: int, fta: int, tov: int) -> Optional[float]:
        """
        Calculate Points Per Possession (PPP)
        
        Measures scoring efficiency per possession used.
        
        Formula: PPP = PTS / (FGA + 0.44 * FTA + TOV)
        
        Args:
            points: Points scored
            fga: Field goal attempts
            fta: Free throw attempts
            tov: Turnovers
        
        Returns:
            PPP as a decimal (e.g., 1.05), or None if undefined
        """
        possessions = fga + 0.44 * fta + tov
        
        if possessions == 0:
            return None
        
        return points / possessions
    
    @staticmethod
    def estimate_possessions(fga: int, fta: int, oreb: int, tov: int) -> float:
        """
        Estimate number of possessions (Dean Oliver formula)
        
        Formula: Poss = FGA + 0.44 * FTA - ORB + TOV
        
        Args:
            fga: Field goal attempts
            fta: Free throw attempts
            oreb: Offensive rebounds
            tov: Turnovers
        
        Returns:
            Estimated possessions
        """
        return fga + 0.44 * fta - oreb + tov
    
    @staticmethod
    def pace(team_possessions: float, opponent_possessions: float, mp: float) -> Optional[float]:
        """
        Calculate Pace (possessions per 48 minutes)
        
        Formula: Pace = 48 * ((TmPoss + OppPoss) / (2 * (MP / 5)))
        
        Args:
            team_possessions: Team possessions
            opponent_possessions: Opponent possessions
            mp: Minutes played (usually 240 for full game)
        
        Returns:
            Pace (possessions per 48 minutes), or None if undefined
        """
        if mp == 0:
            return None
        
        return 48 * ((team_possessions + opponent_possessions) / (2 * (mp / 5)))
    
    @staticmethod
    def four_factors(
        efg_pct: float,
        tov_pct: float,
        orb_pct: float,
        ftr: float,
        weights: Tuple[float, float, float, float] = (0.40, 0.25, 0.20, 0.15)
    ) -> float:
        """
        Calculate Dean Oliver's Four Factors score
        
        The Four Factors of Basketball Success (in order of importance):
        1. Shooting (eFG%) - 40% weight
        2. Turnovers (TOV%) - 25% weight  
        3. Rebounding (ORB%) - 20% weight
        4. Free Throws (FTr) - 15% weight
        
        Args:
            efg_pct: Effective Field Goal Percentage
            tov_pct: Turnover Percentage (lower is better, so invert)
            orb_pct: Offensive Rebound Percentage
            ftr: Free Throw Rate
            weights: Tuple of weights for (eFG%, TOV%, ORB%, FTr)
        
        Returns:
            Weighted Four Factors score
        """
        # Invert TOV% since lower is better
        tov_factor = 1 - (tov_pct / 100) if tov_pct else 0
        
        score = (
            weights[0] * efg_pct +
            weights[1] * tov_factor +
            weights[2] * (orb_pct / 100 if orb_pct else 0) +
            weights[3] * ftr
        )
        
        return score
    
    @classmethod
    def calculate_all_player_stats(cls, box_score: PlayerBoxScore) -> Dict[str, Optional[float]]:
        """
        Calculate all available advanced stats for a player
        
        Args:
            box_score: PlayerBoxScore dataclass with all necessary statistics
        
        Returns:
            Dictionary of advanced statistics
        """
        stats = {}
        
        # Shooting efficiency
        stats['ts_pct'] = cls.true_shooting_percentage(
            box_score.points,
            box_score.field_goals_attempted,
            box_score.free_throws_attempted
        )
        
        stats['efg_pct'] = cls.effective_field_goal_percentage(
            box_score.field_goals_made,
            box_score.three_pointers_made,
            box_score.field_goals_attempted
        )
        
        stats['ftr'] = cls.free_throw_rate(
            box_score.free_throws_attempted,
            box_score.field_goals_attempted
        )
        
        # Scoring efficiency
        stats['ppp'] = cls.points_per_possession(
            box_score.points,
            box_score.field_goals_attempted,
            box_score.free_throws_attempted,
            box_score.turnovers
        )
        
        # Possession stats (require team data)
        if all([box_score.team_field_goals_attempted, box_score.team_free_throws_attempted, box_score.team_turnovers]):
            stats['usg_pct'] = cls.usage_percentage(
                box_score.field_goals_attempted,
                box_score.free_throws_attempted,
                box_score.turnovers,
                box_score.minutes_played,
                box_score.team_field_goals_attempted,
                box_score.team_free_throws_attempted,
                box_score.team_turnovers,
                box_score.team_minutes
            )
        else:
            stats['usg_pct'] = None
        
        stats['tov_pct'] = cls.turnover_percentage(
            box_score.field_goals_attempted,
            box_score.free_throws_attempted,
            box_score.turnovers
        )
        
        # Playmaking
        if all([box_score.team_field_goals_made, box_score.team_minutes]):
            stats['ast_pct'] = cls.assist_percentage(
                box_score.assists,
                box_score.minutes_played,
                box_score.team_field_goals_made,
                box_score.team_minutes,
                box_score.field_goals_made
            )
        else:
            stats['ast_pct'] = None
        
        # Rebounding
        if all([box_score.team_rebounds, box_score.opponent_rebounds]):
            stats['reb_pct'] = cls.rebound_percentage(
                box_score.rebounds,
                box_score.minutes_played,
                box_score.team_rebounds,
                box_score.opponent_rebounds,
                box_score.team_minutes
            )
        else:
            stats['reb_pct'] = None
        
        if all([box_score.team_offensive_rebounds, box_score.opponent_defensive_rebounds]):
            stats['oreb_pct'] = cls.offensive_rebound_percentage(
                box_score.offensive_rebounds,
                box_score.minutes_played,
                box_score.team_offensive_rebounds,
                box_score.opponent_defensive_rebounds,
                box_score.team_minutes
            )
        else:
            stats['oreb_pct'] = None
        
        if all([box_score.team_defensive_rebounds, box_score.opponent_offensive_rebounds]):
            stats['dreb_pct'] = cls.defensive_rebound_percentage(
                box_score.defensive_rebounds,
                box_score.minutes_played,
                box_score.team_defensive_rebounds,
                box_score.opponent_offensive_rebounds,
                box_score.team_minutes
            )
        else:
            stats['dreb_pct'] = None
        
        # Defense (require opponent data)
        if box_score.opponent_field_goals_attempted:
            opponent_possessions = cls.estimate_possessions(
                box_score.opponent_field_goals_attempted,
                0,  # We don't typically have opponent FTA at player level
                box_score.opponent_offensive_rebounds or 0,
                0   # We don't have opponent turnovers at player level
            )
            
            stats['stl_pct'] = cls.steal_percentage(
                box_score.steals,
                box_score.minutes_played,
                int(opponent_possessions),
                box_score.team_minutes
            )
            
            if box_score.opponent_three_pointers_attempted:
                stats['blk_pct'] = cls.block_percentage(
                    box_score.blocks,
                    box_score.minutes_played,
                    box_score.opponent_field_goals_attempted,
                    box_score.opponent_three_pointers_attempted,
                    box_score.team_minutes
                )
            else:
                stats['blk_pct'] = None
        else:
            stats['stl_pct'] = None
            stats['blk_pct'] = None
        
        return stats
    
    @classmethod
    def calculate_all_team_stats(cls, team: TeamBoxScore, opponent: TeamBoxScore) -> Dict[str, Optional[float]]:
        """
        Calculate all available advanced stats for a team
        
        Args:
            team: TeamBoxScore for the team
            opponent: TeamBoxScore for the opponent
        
        Returns:
            Dictionary of advanced team statistics
        """
        stats = {}
        
        # Shooting efficiency
        stats['ts_pct'] = cls.true_shooting_percentage(
            team.points,
            team.field_goals_attempted,
            team.free_throws_attempted
        )
        
        stats['efg_pct'] = cls.effective_field_goal_percentage(
            team.field_goals_made,
            team.three_pointers_made,
            team.field_goals_attempted
        )
        
        stats['ftr'] = cls.free_throw_rate(
            team.free_throws_attempted,
            team.field_goals_attempted
        )
        
        # Possessions and pace
        team_poss = cls.estimate_possessions(
            team.field_goals_attempted,
            team.free_throws_attempted,
            team.offensive_rebounds,
            team.turnovers
        )
        
        opp_poss = cls.estimate_possessions(
            opponent.field_goals_attempted,
            opponent.free_throws_attempted,
            opponent.offensive_rebounds,
            opponent.turnovers
        )
        
        stats['possessions'] = team_poss
        stats['pace'] = cls.pace(team_poss, opp_poss, team.minutes_played)
        
        # Efficiency
        stats['ppp'] = cls.points_per_possession(
            team.points,
            team.field_goals_attempted,
            team.free_throws_attempted,
            team.turnovers
        )
        
        # Turnover rate
        stats['tov_pct'] = cls.turnover_percentage(
            team.field_goals_attempted,
            team.free_throws_attempted,
            team.turnovers
        )
        
        # Rebounding
        total_rebounds = team.rebounds + opponent.rebounds
        if total_rebounds > 0:
            stats['oreb_pct'] = 100 * team.offensive_rebounds / (team.offensive_rebounds + opponent.defensive_rebounds)
            stats['dreb_pct'] = 100 * team.defensive_rebounds / (team.defensive_rebounds + opponent.offensive_rebounds)
            stats['reb_pct'] = 100 * team.rebounds / total_rebounds
        else:
            stats['oreb_pct'] = None
            stats['dreb_pct'] = None
            stats['reb_pct'] = None
        
        # Offensive and Defensive Ratings (points per 100 possessions)
        if team_poss > 0:
            stats['ortg'] = 100 * team.points / team_poss
        else:
            stats['ortg'] = None
        
        if opp_poss > 0:
            stats['drtg'] = 100 * opponent.points / opp_poss
        else:
            stats['drtg'] = None
        
        # Net Rating
        if stats['ortg'] and stats['drtg']:
            stats['net_rtg'] = stats['ortg'] - stats['drtg']
        else:
            stats['net_rtg'] = None
        
        # Four Factors
        if all([stats['efg_pct'], stats['tov_pct'], stats['oreb_pct'], stats['ftr']]):
            stats['four_factors'] = cls.four_factors(
                stats['efg_pct'],
                stats['tov_pct'],
                stats['oreb_pct'],
                stats['ftr']
            )
        else:
            stats['four_factors'] = None
        
        return stats

