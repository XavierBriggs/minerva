"""
Database Models

SQLAlchemy ORM models following ESPN data structure.
"""

from datetime import datetime
from typing import Optional
from sqlalchemy import (
    Column, Integer, String, Float, DateTime, Date, ForeignKey, 
    Boolean, Text, UniqueConstraint, Index, Numeric
)
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import relationship

Base = declarative_base()


class Season(Base):
    """NBA Season - Top level hierarchy"""
    __tablename__ = "seasons"
    
    season_id = Column(String(10), primary_key=True)  # e.g., "2024-25"
    season_name = Column(String(50))  # e.g., "2024-25 NBA Season"
    start_date = Column(Date, nullable=False)
    end_date = Column(Date, nullable=False)
    playoff_start_date = Column(Date)
    finals_start_date = Column(Date)
    champion_team_id = Column(Integer, ForeignKey("teams.team_id"), nullable=True)
    is_complete = Column(Boolean, default=False)
    total_games = Column(Integer)  # Regular season games expected
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    
    # Relationships
    champion = relationship("Team", foreign_keys=[champion_team_id])
    games = relationship("Game", back_populates="season")
    player_seasons = relationship("PlayerSeason", back_populates="season")
    
    def __repr__(self):
        status = "Complete" if self.is_complete else "In Progress"
        return f"<Season {self.season_id} - {status}>"


class Team(Base):
    """NBA Team"""
    __tablename__ = "teams"
    
    team_id = Column(Integer, primary_key=True, autoincrement=True)
    team_abbreviation = Column(String(10), unique=True, nullable=False, index=True)
    team_name = Column(String(100), nullable=False)
    team_city = Column(String(100))
    team_full_name = Column(String(150))
    conference = Column(String(10))  # East or West
    division = Column(String(50))
    espn_team_id = Column(String(20), unique=True, index=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    
    # Relationships
    home_games = relationship("Game", foreign_keys="Game.home_team_id", back_populates="home_team")
    away_games = relationship("Game", foreign_keys="Game.away_team_id", back_populates="away_team")
    players = relationship("Player", back_populates="team")
    
    def __repr__(self):
        return f"<Team {self.team_abbreviation}: {self.team_full_name}>"


class Player(Base):
    """NBA Player"""
    __tablename__ = "players"
    
    player_id = Column(Integer, primary_key=True, autoincrement=True)
    player_name = Column(String(150), nullable=False, index=True)
    espn_player_id = Column(String(20), unique=True, index=True)
    nba_api_id = Column(String(20), unique=True, index=True)
    team_id = Column(Integer, ForeignKey("teams.team_id"), nullable=True)
    position = Column(String(10))
    jersey_number = Column(String(5))
    height = Column(String(10))
    weight = Column(Integer)
    birth_date = Column(Date)
    college = Column(String(100))
    is_active = Column(Boolean, default=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    
    # Relationships
    team = relationship("Team", back_populates="players")
    game_stats = relationship("PlayerGameStats", back_populates="player")
    player_seasons = relationship("PlayerSeason", back_populates="player")
    
    def __repr__(self):
        return f"<Player {self.player_name} ({self.team.team_abbreviation if self.team else 'FA'})>"


class PlayerSeason(Base):
    """Player participation in a specific season"""
    __tablename__ = "player_seasons"
    
    id = Column(Integer, primary_key=True, autoincrement=True)
    player_id = Column(Integer, ForeignKey("players.player_id"), nullable=False, index=True)
    season_id = Column(String(10), ForeignKey("seasons.season_id"), nullable=False, index=True)
    team_id = Column(Integer, ForeignKey("teams.team_id"), nullable=True)  # Can be NULL for free agents
    
    # Season-specific info (can change between seasons)
    position = Column(String(10))
    jersey_number = Column(String(5))
    was_active = Column(Boolean, default=True)  # Was player active this season?
    games_played = Column(Integer, default=0)  # Total games in this season
    
    # Season stats summary (aggregated from player_game_stats)
    season_ppg = Column(Float)  # Points per game
    season_rpg = Column(Float)  # Rebounds per game
    season_apg = Column(Float)  # Assists per game
    
    # Metadata
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    
    # Relationships
    player = relationship("Player", back_populates="player_seasons")
    season = relationship("Season", back_populates="player_seasons")
    team = relationship("Team")
    
    __table_args__ = (
        UniqueConstraint('player_id', 'season_id', name='uq_player_season'),
        Index('idx_season_player', 'season_id', 'player_id'),
    )
    
    def __repr__(self):
        team = self.team.team_abbreviation if self.team else "FA"
        return f"<PlayerSeason {self.player.player_name if self.player else '?'} {self.season_id} ({team})>"


class Game(Base):
    """NBA Game"""
    __tablename__ = "games"
    
    id = Column(Integer, primary_key=True, autoincrement=True)
    game_id = Column(String(50), unique=True, nullable=False, index=True)  # ESPN game ID
    season_id = Column(String(10), ForeignKey("seasons.season_id"), nullable=False, index=True)  # e.g., "2025-26"
    season_type = Column(String(20), default="Regular Season")  # Regular Season, Playoffs, etc.
    game_date = Column(Date, nullable=False, index=True)
    game_time = Column(DateTime)
    home_team_id = Column(Integer, ForeignKey("teams.team_id"), nullable=False)
    away_team_id = Column(Integer, ForeignKey("teams.team_id"), nullable=False)
    home_score = Column(Integer)
    away_score = Column(Integer)
    game_status = Column(String(20))  # Scheduled, In Progress, Final
    venue = Column(String(150))
    attendance = Column(Integer)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    
    # Relationships
    season = relationship("Season", back_populates="games", foreign_keys=[season_id])
    home_team = relationship("Team", foreign_keys=[home_team_id], back_populates="home_games")
    away_team = relationship("Team", foreign_keys=[away_team_id], back_populates="away_games")
    player_stats = relationship("PlayerGameStats", back_populates="game")
    team_odds = relationship("TeamOdds", back_populates="game")
    player_props = relationship("PlayerPropOdds", back_populates="game")
    
    __table_args__ = (
        Index('idx_game_date_teams', 'game_date', 'home_team_id', 'away_team_id'),
    )
    
    def __repr__(self):
        return f"<Game {self.game_id}: {self.away_team.team_abbreviation if self.away_team else '?'} @ {self.home_team.team_abbreviation if self.home_team else '?'}>"


class PlayerGameStats(Base):
    """Player statistics for a specific game"""
    __tablename__ = "player_game_stats"
    
    id = Column(Integer, primary_key=True, autoincrement=True)
    game_id = Column(String(50), ForeignKey("games.game_id"), nullable=False, index=True)
    player_id = Column(Integer, ForeignKey("players.player_id"), nullable=False, index=True)
    team_id = Column(Integer, ForeignKey("teams.team_id"), nullable=False)
    
    # Basic stats
    points = Column(Integer, default=0)
    rebounds = Column(Integer, default=0)
    assists = Column(Integer, default=0)
    steals = Column(Integer, default=0)
    blocks = Column(Integer, default=0)
    turnovers = Column(Integer, default=0)
    
    # Shooting stats
    field_goals_made = Column(Integer, default=0)
    field_goals_attempted = Column(Integer, default=0)
    three_pointers_made = Column(Integer, default=0)
    three_pointers_attempted = Column(Integer, default=0)
    free_throws_made = Column(Integer, default=0)
    free_throws_attempted = Column(Integer, default=0)
    
    # Additional stats
    offensive_rebounds = Column(Integer, default=0)
    defensive_rebounds = Column(Integer, default=0)
    personal_fouls = Column(Integer, default=0)
    minutes_played = Column(Float, default=0.0)
    plus_minus = Column(Integer)
    
    # Composite stats
    points_rebounds_assists = Column(Integer, default=0)  # PRA
    
    # Advanced statistics
    true_shooting_pct = Column(Float)  # TS%
    effective_fg_pct = Column(Float)  # eFG%
    usage_pct = Column(Float)  # USG%
    assist_pct = Column(Float)  # AST%
    rebound_pct = Column(Float)  # REB%
    offensive_rebound_pct = Column(Float)  # ORB%
    defensive_rebound_pct = Column(Float)  # DRB%
    turnover_pct = Column(Float)  # TOV%
    steal_pct = Column(Float)  # STL%
    block_pct = Column(Float)  # BLK%
    free_throw_rate = Column(Float)  # FTr
    points_per_possession = Column(Float)  # PPP
    
    # Metadata
    starter = Column(Boolean, default=False)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    
    # Relationships
    game = relationship("Game", back_populates="player_stats")
    player = relationship("Player", back_populates="game_stats")
    team = relationship("Team")
    
    __table_args__ = (
        UniqueConstraint('game_id', 'player_id', name='uq_game_player'),
        Index('idx_player_game_date', 'player_id', 'game_id'),
    )
    
    def __repr__(self):
        return f"<PlayerGameStats {self.player.player_name if self.player else '?'}: {self.points}pts {self.rebounds}reb {self.assists}ast>"


class TeamGameStats(Base):
    """Team statistics for a specific game"""
    __tablename__ = "team_game_stats"
    
    id = Column(Integer, primary_key=True, autoincrement=True)
    game_id = Column(String(50), ForeignKey("games.game_id"), nullable=False, index=True)
    team_id = Column(Integer, ForeignKey("teams.team_id"), nullable=False, index=True)
    is_home = Column(Boolean, nullable=False)
    
    # Basic stats
    points = Column(Integer, default=0)
    field_goals_made = Column(Integer, default=0)
    field_goals_attempted = Column(Integer, default=0)
    three_pointers_made = Column(Integer, default=0)
    three_pointers_attempted = Column(Integer, default=0)
    free_throws_made = Column(Integer, default=0)
    free_throws_attempted = Column(Integer, default=0)
    offensive_rebounds = Column(Integer, default=0)
    defensive_rebounds = Column(Integer, default=0)
    rebounds = Column(Integer, default=0)
    assists = Column(Integer, default=0)
    steals = Column(Integer, default=0)
    blocks = Column(Integer, default=0)
    turnovers = Column(Integer, default=0)
    personal_fouls = Column(Integer, default=0)
    
    # Advanced statistics
    true_shooting_pct = Column(Float)  # TS%
    effective_fg_pct = Column(Float)  # eFG%
    turnover_pct = Column(Float)  # TOV%
    offensive_rebound_pct = Column(Float)  # ORB%
    defensive_rebound_pct = Column(Float)  # DRB%
    free_throw_rate = Column(Float)  # FTr
    possessions = Column(Float)  # Estimated possessions
    pace = Column(Float)  # Pace (possessions per 48 min)
    offensive_rating = Column(Float)  # ORtg (points per 100 possessions)
    defensive_rating = Column(Float)  # DRtg (points allowed per 100 possessions)
    net_rating = Column(Float)  # Net Rating (ORtg - DRtg)
    points_per_possession = Column(Float)  # PPP
    four_factors_score = Column(Float)  # Dean Oliver's Four Factors
    
    # Metadata
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    
    __table_args__ = (
        UniqueConstraint('game_id', 'team_id', name='uq_game_team'),
        Index('idx_team_game', 'team_id', 'game_id'),
    )
    
    def __repr__(self):
        return f"<TeamGameStats Game:{self.game_id} Team:{self.team_id} {self.points}pts>"


class TeamOdds(Base):
    """Team betting odds for a game"""
    __tablename__ = "team_odds"
    
    id = Column(Integer, primary_key=True, autoincrement=True)
    game_id = Column(String(50), ForeignKey("games.game_id"), nullable=False, index=True)
    team_id = Column(Integer, ForeignKey("teams.team_id"), nullable=False)
    bookmaker = Column(String(100), nullable=False, index=True)
    
    # Moneyline
    moneyline = Column(Integer)  # American odds format (e.g., -150, +120)
    
    # Spread
    spread_points = Column(Float)  # e.g., -5.5
    spread_odds = Column(Integer)  # American odds for spread
    
    # Totals (Over/Under)
    total_points = Column(Float)  # e.g., 215.5
    over_odds = Column(Integer)
    under_odds = Column(Integer)
    
    # Metadata
    last_update = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    is_active = Column(Boolean, default=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    
    # Relationships
    game = relationship("Game", back_populates="team_odds")
    
    __table_args__ = (
        Index('idx_game_bookmaker', 'game_id', 'bookmaker'),
    )
    
    def __repr__(self):
        return f"<TeamOdds {self.bookmaker}: ML={self.moneyline}, Spread={self.spread_points}>"


class PlayerPropOdds(Base):
    """Player prop betting odds"""
    __tablename__ = "player_prop_odds"
    
    id = Column(Integer, primary_key=True, autoincrement=True)
    game_id = Column(String(50), ForeignKey("games.game_id"), nullable=False, index=True)
    player_id = Column(Integer, ForeignKey("players.player_id"), nullable=False, index=True)
    bookmaker = Column(String(100), nullable=False)
    
    # Prop details
    prop_type = Column(String(50), nullable=False, index=True)  # points, rebounds, assists, etc.
    line = Column(Float, nullable=False)  # The prop line (e.g., 25.5 points)
    over_odds = Column(Integer)  # American odds for over
    under_odds = Column(Integer)  # American odds for under
    
    # Metadata
    last_update = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    is_active = Column(Boolean, default=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    
    # Relationships
    game = relationship("Game", back_populates="player_props")
    
    __table_args__ = (
        Index('idx_player_prop_type', 'player_id', 'prop_type'),
        Index('idx_game_player_bookmaker', 'game_id', 'player_id', 'bookmaker'),
    )
    
    def __repr__(self):
        return f"<PlayerPropOdds {self.prop_type}: {self.line} ({self.bookmaker})>"


class OddsAPIMapping(Base):
    """Mapping between ESPN and Odds API identifiers"""
    __tablename__ = "odds_api_mappings"
    
    id = Column(Integer, primary_key=True, autoincrement=True)
    espn_game_id = Column(String(50), nullable=False, index=True)
    odds_api_event_id = Column(String(100), nullable=False, index=True)
    espn_team_id = Column(String(20), index=True)
    odds_api_team_name = Column(String(100))
    created_at = Column(DateTime, default=datetime.utcnow)
    
    def __repr__(self):
        return f"<OddsAPIMapping ESPN:{self.espn_game_id} -> Odds:{self.odds_api_event_id}>"

