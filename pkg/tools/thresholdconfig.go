package tools

import (
	"encoding/json"
	"fmt"
	"os"
)

var (
	defaultThresholdConfig = ThresholdConfig{
		ThresholdSteps: []float64{0.07, 0.08, 0.09, 0.1, 0.125, 0.15, 0.2, 0.25, 0.3, 0.35, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
		MinValid:       500,
		Data:           []int{8700, 4800, 3500, 2550, 1850, 1387, 992, 840, 750, 700, 655, 620, 590, 570, 553, 549, 548},
	}
)

func NewThresholdConfig() *ThresholdConfig {
	return &defaultThresholdConfig
}

func NewThresholdConfigFromJson(fileName string) (*ThresholdConfig, error) {
	content, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("unable to read content from %s file: %w", fileName, err)
	}
	var ft ThresholdConfig
	err = json.Unmarshal(content, &ft)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal json content from %s file: %w", fileName, err)
	}
	return &ft, nil
}

type ThresholdConfig struct {
	ThresholdSteps []float64 `json:"threshold_steps"`
	MinValid       int       `json:"min_valid"`
	Data           []int     `json:"data"`
}

func (tc *ThresholdConfig) ValueOf(pwm int) float64 {
	if pwm < tc.MinValid || pwm > tc.Data[0] {
		return 0.
	}
	if pwm == tc.Data[0] {
		return tc.ThresholdSteps[0]
	}

	if pwm < tc.Data[len(tc.Data)-1] && pwm >= tc.MinValid {
		return 1.
	}
	// search column index
	var idx int
	// Start loop at 1 because first column should be skipped
	for i := 1; i < len(tc.ThresholdSteps); i++ {
		if pwm == tc.Data[i] {
			return tc.ThresholdSteps[i]
		}
		if pwm > tc.Data[i] {
			idx = i - 1
			break
		}
	}

	return tc.ThresholdSteps[idx] - (tc.ThresholdSteps[idx]-tc.ThresholdSteps[idx+1])/2.
}
