package recommender

import (
	"testing"

	"github.com/opscart/k8s-cost-optimizer/pkg/analyzer"
)

// Test pattern-based buffer adjustments
func TestAdjustSafetyBufferForPattern(t *testing.T) {
	tests := []struct {
		name        string
		baseBuffer  float64
		cpuPattern  analyzer.UsagePattern
		expectedMin float64
		expectedMax float64
		description string
	}{
		{
			name:       "Steady pattern reduces buffer",
			baseBuffer: 2.0,
			cpuPattern: analyzer.UsagePattern{
				Type:      "steady",
				Variation: 0.1,
			},
			expectedMin: 1.7, // 2.0 * 0.90 = 1.8, but might be clamped
			expectedMax: 1.9,
			description: "Steady workloads should get -10% buffer reduction",
		},
		{
			name:       "Spiky pattern increases buffer",
			baseBuffer: 1.5,
			cpuPattern: analyzer.UsagePattern{
				Type:      "spiky",
				Variation: 0.6,
			},
			expectedMin: 1.7, // 1.5 * 1.15 = 1.725, plus CV adjustment
			expectedMax: 2.0,
			description: "Spiky workloads should get +15% buffer increase",
		},
		{
			name:       "Highly variable pattern increases buffer significantly",
			baseBuffer: 1.5,
			cpuPattern: analyzer.UsagePattern{
				Type:      "highly-variable",
				Variation: 1.2,
			},
			expectedMin: 2.0, // 1.5 * 1.25 * 1.10 = 2.0625
			expectedMax: 2.2,
			description: "Highly variable workloads need maximum buffer",
		},
		{
			name:       "Minimum buffer enforcement",
			baseBuffer: 1.0,
			cpuPattern: analyzer.UsagePattern{
				Type:      "steady",
				Variation: 0.05,
			},
			expectedMin: 1.2, // Minimum is 1.2x
			expectedMax: 1.2,
			description: "Buffer should never go below 1.2x",
		},
		{
			name:       "Maximum buffer enforcement",
			baseBuffer: 2.8,
			cpuPattern: analyzer.UsagePattern{
				Type:      "highly-variable",
				Variation: 1.5,
			},
			expectedMin: 3.0, // Maximum is 3.0x
			expectedMax: 3.0,
			description: "Buffer should never exceed 3.0x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adjustSafetyBufferForPattern(tt.baseBuffer, tt.cpuPattern, analyzer.UsagePattern{})

			if result < tt.expectedMin || result > tt.expectedMax {
				t.Errorf("%s: got %.2f, expected between %.2f and %.2f\n%s",
					tt.name, result, tt.expectedMin, tt.expectedMax, tt.description)
			}
		})
	}
}

// Test growth trend adjustments
func TestAdjustForGrowthTrend(t *testing.T) {
	tests := []struct {
		name        string
		baseValue   int64
		growth      analyzer.GrowthTrend
		expectedMin int64
		expectedMax int64
		description string
	}{
		{
			name:      "No growth - no adjustment",
			baseValue: 1000,
			growth: analyzer.GrowthTrend{
				IsGrowing:       false,
				RatePerMonth:    2.0,
				Predicted3Month: 1060,
			},
			expectedMin: 1000,
			expectedMax: 1000,
			description: "Non-growing workloads should not get growth buffer",
		},
		{
			name:      "Low growth (<5%/month) - no adjustment",
			baseValue: 1000,
			growth: analyzer.GrowthTrend{
				IsGrowing:       true,
				RatePerMonth:    3.0,
				Predicted3Month: 1090,
			},
			expectedMin: 1000,
			expectedMax: 1000,
			description: "Growth below 5%/month threshold should not add buffer",
		},
		{
			name:      "Moderate growth (10%/month) - add buffer",
			baseValue: 1000,
			growth: analyzer.GrowthTrend{
				IsGrowing:       true,
				RatePerMonth:    10.0,
				Predicted3Month: 1300,
			},
			expectedMin: 1600, // Base 1000 + (1300 * 0.5) = 1650
			expectedMax: 1700,
			description: "Growing workloads should get 50% of predicted 3-month value added",
		},
		{
			name:      "High growth (50%/month) - significant buffer",
			baseValue: 500,
			growth: analyzer.GrowthTrend{
				IsGrowing:       true,
				RatePerMonth:    50.0,
				Predicted3Month: 1250,
			},
			expectedMin: 1100, // Base 500 + (1250 * 0.5) = 1125
			expectedMax: 1150,
			description: "Rapidly growing workloads need substantial buffer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adjustForGrowthTrend(tt.baseValue, tt.growth)

			if result < tt.expectedMin || result > tt.expectedMax {
				t.Errorf("%s: got %d, expected between %d and %d\n%s",
					tt.name, result, tt.expectedMin, tt.expectedMax, tt.description)
			}
		})
	}
}

// Test confidence calculation
func TestCalculateConfidence(t *testing.T) {
	tests := []struct {
		name              string
		dataQuality       float64
		hasSufficientData bool
		patternType       string
		expected          string
		description       string
	}{
		{
			name:              "Insufficient data always LOW",
			dataQuality:       0.9,
			hasSufficientData: false,
			patternType:       "steady",
			expected:          "LOW",
			description:       "Without sufficient data, confidence must be LOW",
		},
		{
			name:              "High quality + steady = HIGH confidence",
			dataQuality:       0.85,
			hasSufficientData: true,
			patternType:       "steady",
			expected:          "HIGH",
			description:       "Best case: high quality data with steady pattern",
		},
		{
			name:              "High quality + variable = MEDIUM confidence",
			dataQuality:       0.85,
			hasSufficientData: true,
			patternType:       "spiky",
			expected:          "MEDIUM",
			description:       "Variable patterns reduce confidence even with good data",
		},
		{
			name:              "Medium quality = MEDIUM confidence",
			dataQuality:       0.7,
			hasSufficientData: true,
			patternType:       "steady",
			expected:          "MEDIUM",
			description:       "Acceptable quality gives medium confidence",
		},
		{
			name:              "Low quality = LOW confidence",
			dataQuality:       0.4,
			hasSufficientData: true,
			patternType:       "steady",
			expected:          "LOW",
			description:       "Poor quality data cannot give high confidence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateConfidence(tt.dataQuality, tt.hasSufficientData, tt.patternType)

			if result != tt.expected {
				t.Errorf("%s: got %s, expected %s\n%s",
					tt.name, result, tt.expected, tt.description)
			}
		})
	}
}

// Test pattern info building
func TestBuildPatternInfo(t *testing.T) {
	tests := []struct {
		name        string
		cpuPattern  analyzer.UsagePattern
		memPattern  analyzer.UsagePattern
		cpuGrowth   analyzer.GrowthTrend
		expected    string
		description string
	}{
		{
			name: "Steady pattern without growth",
			cpuPattern: analyzer.UsagePattern{
				Type: "steady",
			},
			memPattern: analyzer.UsagePattern{
				Type: "steady",
			},
			cpuGrowth: analyzer.GrowthTrend{
				IsGrowing:    false,
				RatePerMonth: 2.0,
			},
			expected:    "CPU: steady",
			description: "Should show pattern type",
		},
		{
			name: "Moderate pattern with significant growth",
			cpuPattern: analyzer.UsagePattern{
				Type: "moderate",
			},
			memPattern: analyzer.UsagePattern{
				Type: "moderate",
			},
			cpuGrowth: analyzer.GrowthTrend{
				IsGrowing:    true,
				RatePerMonth: 15.0,
			},
			expected:    "CPU: moderate, Growing 15%/mo",
			description: "Should show pattern and growth",
		},
		{
			name:        "No pattern data",
			cpuPattern:  analyzer.UsagePattern{},
			memPattern:  analyzer.UsagePattern{},
			cpuGrowth:   analyzer.GrowthTrend{},
			expected:    "Insufficient data",
			description: "Should indicate insufficient data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPatternInfo(tt.cpuPattern, tt.memPattern, tt.cpuGrowth)

			if result != tt.expected {
				t.Errorf("%s: got '%s', expected '%s'\n%s",
					tt.name, result, tt.expected, tt.description)
			}
		})
	}
}
