import { useState, useEffect, useRef } from 'react';
import { CurrentPickState, DraftStatus } from '../types/draft';

export function useDraftTimer(currentPick?: CurrentPickState, status?: DraftStatus) {
  const [timeRemaining, setTimeRemaining] = useState(0);
  const [isPaused, setIsPaused] = useState(false);
  const timerRef = useRef<NodeJS.Timeout | null>(null);
  const deadlineRef = useRef<number | null>(null);

  // Clear timer on cleanup
  useEffect(() => {
    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current);
      }
    };
  }, []);

  // Main timer logic
  useEffect(() => {
    console.log('Timer hook: currentPick=', currentPick, 'status=', status);
    
    // Clear existing timer
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }

    // Reset if no current pick or draft not in progress
    if (!currentPick || status !== DraftStatus.IN_PROGRESS) {
      console.log('Timer hook: No current pick or not in progress, setting time to 0');
      setTimeRemaining(0);
      deadlineRef.current = null;
      return;
    }

    // Calculate deadline from server-provided timestamp
    const startTime = new Date(currentPick.startedAt).getTime();
    const deadline = startTime + (currentPick.timePerPickSec * 1000);
    deadlineRef.current = deadline;
    
    console.log('Timer hook: startTime=', new Date(currentPick.startedAt), 'timePerPickSec=', currentPick.timePerPickSec, 'deadline=', new Date(deadline));

    // Timer tick function
    const tick = () => {
      if (!deadlineRef.current || isPaused) return;
      
      const now = Date.now();
      const remaining = Math.max(0, Math.ceil((deadlineRef.current - now) / 1000));
      setTimeRemaining(remaining);
      
      // Stop when time expires
      if (remaining === 0 && timerRef.current) {
        clearInterval(timerRef.current);
        timerRef.current = null;
      }
    };

    // Start timer
    tick(); // Run immediately
    timerRef.current = setInterval(tick, 1000);

  }, [currentPick, status, isPaused]);

  // Handle pause/resume
  const pauseTimer = () => setIsPaused(true);
  const resumeTimer = () => setIsPaused(false);

  // Reset timer with new server data
  const resetTimer = (newStartedAt: string, newTimePerPickSec: number) => {
    const startTime = new Date(newStartedAt).getTime();
    const deadline = startTime + (newTimePerPickSec * 1000);
    deadlineRef.current = deadline;
    setIsPaused(false);
  };

  return {
    timeRemaining,
    isPaused,
    pauseTimer,
    resumeTimer,
    resetTimer,
  };
}