package gateway

// Simple Timer Synchronization - No complex clock sync needed
//
// Strategy: Send timer duration to client, let client count down
// - PickStarted event includes time_per_pick_sec
// - Client starts countdown from that duration
// - Server timeout is authoritative for auto-pick
// - Client timer is just visual feedback

// Timer approach:
// 1. Server sends: {"type": "PickStarted", "data": {"time_per_pick_sec": 60}}
// 2. Client counts down: 60, 59, 58, 57... 0
// 3. Server handles real timeout and auto-pick
// 4. On reconnect: send time_remaining_sec in state sync

// No complex clock synchronization needed - keep it simple!
