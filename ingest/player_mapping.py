"""
Player ID Mapping

Maps player IDs between ESPN and internal database.
"""

from typing import Dict, Optional
from sqlalchemy.orm import Session

from Gringotts.store.models import Player
from src.utils.logger import logger


class PlayerIDMapper:
    """Maps ESPN player IDs to database player IDs"""
    
    def __init__(self):
        """Initialize player mapper"""
        self.espn_to_db_cache: Dict[str, int] = {}
        self.name_to_db_cache: Dict[str, int] = {}
    
    def load_cache_from_db(self, session: Optional[Session] = None):
        """
        Load existing player mappings into cache
        
        Args:
            session: Database session (optional)
        """
        if session:
            players = session.query(Player).all()
            for player in players:
                if player.espn_player_id:
                    self.espn_to_db_cache[player.espn_player_id] = player.player_id
                self.name_to_db_cache[player.player_name.lower()] = player.player_id
            
            logger.debug(f"Loaded {len(self.espn_to_db_cache)} player mappings from database")
    
    def map_espn_id_to_player(
        self,
        espn_id: str,
        player_name: str,
        team_abbreviation: Optional[str],
        session: Session
    ) -> Optional[int]:
        """
        Map ESPN player ID to database player ID
        
        Args:
            espn_id: ESPN player ID
            player_name: Player name
            team_abbreviation: Team abbreviation
            session: Database session
        
        Returns:
            Database player ID or None
        """
        # Check cache first
        if espn_id in self.espn_to_db_cache:
            return self.espn_to_db_cache[espn_id]
        
        # Look up in database
        player = session.query(Player).filter_by(espn_player_id=espn_id).first()
        
        if player:
            self.espn_to_db_cache[espn_id] = player.player_id
            return player.player_id
        
        # Try to find by name
        player = session.query(Player).filter_by(player_name=player_name).first()
        
        if player:
            # Update ESPN ID
            player.espn_player_id = espn_id
            session.flush()
            self.espn_to_db_cache[espn_id] = player.player_id
            logger.info(f"Mapped ESPN ID {espn_id} to existing player {player_name}")
            return player.player_id
        
        # Create new player
        logger.info(f"Creating new player: {player_name} (ESPN ID: {espn_id})")
        
        # Get team ID
        from Gringotts.store.models import Team
        team = None
        if team_abbreviation:
            team = session.query(Team).filter_by(team_abbreviation=team_abbreviation).first()
        
        new_player = Player(
            player_name=player_name,
            espn_player_id=espn_id,
            team_id=team.team_id if team else None
        )
        session.add(new_player)
        session.flush()
        
        self.espn_to_db_cache[espn_id] = new_player.player_id
        return new_player.player_id
    
    def clear_cache(self):
        """Clear the mapping cache"""
        self.espn_to_db_cache.clear()
        self.name_to_db_cache.clear()

