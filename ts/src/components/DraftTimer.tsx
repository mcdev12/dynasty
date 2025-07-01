import React, { useEffect } from 'react';
import { View, Text, StyleSheet, Animated } from 'react-native';
import { CurrentPickState, DraftStatus } from '../types/draft';
import { useDraftTimer } from '../hooks/useDraftTimer';

interface DraftTimerProps {
  currentPick?: CurrentPickState;
  status: DraftStatus;
  onTimerPause?: () => void;
  onTimerResume?: () => void;
  onTimerReset?: (startedAt: string, timePerPickSec: number) => void;
}

export default function DraftTimer({ currentPick, status, onTimerPause, onTimerResume, onTimerReset }: DraftTimerProps) {
  const { timeRemaining, isPaused, pauseTimer, resumeTimer, resetTimer } = useDraftTimer(currentPick, status);
  const pulseAnim = React.useRef(new Animated.Value(1)).current;

  // Expose timer controls to parent
  useEffect(() => {
    if (onTimerPause) onTimerPause = pauseTimer;
    if (onTimerResume) onTimerResume = resumeTimer; 
    if (onTimerReset) onTimerReset = resetTimer;
  }, [pauseTimer, resumeTimer, resetTimer, onTimerPause, onTimerResume, onTimerReset]);

  useEffect(() => {
    if (timeRemaining > 0 && !isPaused) {
      Animated.loop(
        Animated.sequence([
          Animated.timing(pulseAnim, {
            toValue: 1.2,
            duration: 500,
            useNativeDriver: true,
          }),
          Animated.timing(pulseAnim, {
            toValue: 1,
            duration: 500,
            useNativeDriver: true,
          }),
        ])
      ).start();
    } else {
      pulseAnim.setValue(1);
    }
  }, [timeRemaining, isPaused]);

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const getTimerColor = () => {
    if (isPaused) return '#fbbf24'; // yellow when paused
    if (timeRemaining <= 30) return '#ef4444'; // red when urgent
    if (timeRemaining <= 0) return '#6b7280'; // gray when expired
    return '#3b82f6'; // blue when normal
  };

  return (
    <View style={styles.container}>
      <View style={styles.timerContainer}>
        <Animated.View 
          style={[
            styles.timerDot,
            { backgroundColor: getTimerColor(), transform: [{ scale: pulseAnim }] }
          ]} 
        />
        <Text style={[styles.timerText, { color: getTimerColor() }]}>
          {formatTime(timeRemaining)}
        </Text>
      </View>
      <Text style={styles.title}>Live Draft</Text>
      <View style={styles.placeholder} />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 20,
    backgroundColor: '#0f1419',
  },
  timerContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    flex: 1,
  },
  timerDot: {
    width: 10,
    height: 10,
    borderRadius: 5,
    marginRight: 8,
  },
  timerText: {
    fontSize: 16,
    fontWeight: 'bold',
    fontFamily: 'monospace',
  },
  title: {
    fontSize: 20,
    fontWeight: 'bold',
    color: '#ffffff',
    flex: 1,
    textAlign: 'center',
  },
  placeholder: {
    flex: 1,
  },
});