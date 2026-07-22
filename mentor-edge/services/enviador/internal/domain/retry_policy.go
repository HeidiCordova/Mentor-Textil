package domain

import "time"

type RetryPolicy struct {
	InlineRetries   int
	MaxEventRetries int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
}

func NewDefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		InlineRetries:   3,
		MaxEventRetries: 2880,
		InitialDelay:    2 * time.Second,
		MaxDelay:        2 * time.Minute,
		BackoffFactor:   2.0,
	}
}

func (p *RetryPolicy) GetDelay(attemptNumber int) time.Duration {
	delay := float64(p.InitialDelay)
	for i := 0; i < attemptNumber; i++ {
		delay *= p.BackoffFactor
	}
	if time.Duration(delay) > p.MaxDelay {
		return p.MaxDelay
	}
	return time.Duration(delay)
}

func (p *RetryPolicy) ShouldRetryInline(attemptNumber int) bool {
	return attemptNumber < p.InlineRetries
}

func (p *RetryPolicy) IsEventExhausted(retryCount int) bool {
	return retryCount >= p.MaxEventRetries
}
