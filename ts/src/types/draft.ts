// Import proto-generated types
import { Draft, DraftPick, DraftSettings, DraftStatus, DraftType } from '../genproto/draft/v1/draft_pb';
import { Player } from '../genproto/player/v1/player_pb';
import { PlayerProfile, NFLPlayerProfile } from '../genproto/player/v1/player_profile_pb';

// Export proto types for convenience
export { Draft, DraftPick, DraftSettings, DraftStatus, DraftType, Player, PlayerProfile, NFLPlayerProfile };

// Additional UI-specific types not in protos
export interface Team {
  id: string;
  name: string;
  avatar: string;
  color: string;
}

export interface EnrichedPlayer extends Player {
  rank: number;
  projection?: string;
  position?: string; // Extracted from profile
  nflTeam?: string; // Extracted from teamId
}

export interface CurrentPickState {
  teamId: string;
  round: number;
  pickNumber: number;
  overallPick: number;
  startedAt: string;
  timePerPickSec: number;
}

// Gateway API Response format (what /api/drafts/{id}/state returns)
export interface DraftGatewayStateResponse {
  draft_id: string;
  status: string;
  current_pick?: {
    team_id: string;
    round: number;
    pick: number;
    overall_pick: number;
    started_at: string;
    time_per_pick_sec: number;
  };
  recent_picks: any[] | null;
  total_picks: number;
  completed_picks: number;
  time_remaining_sec: number;
  metadata: {
    draft_type: string;
    league_id: string;
    total_rounds: number;
    total_teams: number;
  };
}

// UI State format (what we use in components) 
export interface DraftUIState {
  // Basic draft info from GetDraft response
  draft_id: string;
  status: DraftStatus;
  metadata: {
    draft_type: DraftType;
    league_id: string;
    total_rounds: number;
    total_teams: number;
  };
  
  // Draft settings from GetDraft
  settings?: DraftSettings;
  
  // Additional UI data (to be populated)
  currentPick?: CurrentPickState;
  teams: Team[];
  picks: DraftPick[];
  availablePlayers: EnrichedPlayer[];
}

export interface DraftEvent {
  type: 'PICK_MADE' | 'PICK_STARTED' | 'PICK_ENDED' | 'PICK_PAUSED' | 'PICK_RESUMED' | 'DRAFT_STARTED' | 'DRAFT_COMPLETED' | 'DRAFT_PAUSED' | 'DRAFT_RESUMED';
  timestamp: string;
  payload: {
    startedAt?: string;
    timePerPickSec?: number;
    teamId?: string;
    round?: number;
    pickNumber?: number;
    overallPick?: number;
    playerId?: string;
    [key: string]: any;
  };
}

export const POSITION_COLORS = {
  QB: '#ef4444',
  RB: '#22c55e',
  WR: '#3b82f6',
  TE: '#eab308',
  K: '#a855f7',
  DEF: '#6b7280',
} as const;