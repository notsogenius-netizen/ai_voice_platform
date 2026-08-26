package rules

// DefaultThresholds returns the built-in thresholds for every rule.
func DefaultThresholds() map[RuleID]Thresholds {
	return map[RuleID]Thresholds{
		MethodsPerFile:     {Minor: 25, Major: 35},
		ReturnsPerFunction: {Minor: 5, Major: 7},
		Complexity:         {Minor: 5, Major: 10},
		LOCPerFile:         {Minor: 250, Major: 400},
		FunctionLength:     {Minor: 20, Major: 30},
		Parameters:         {Minor: 5, Major: 8},
		NestingDepth:       {Minor: 3, Major: 5},
	}
}

// DefaultFailOn is the default quality-gate failure severity.
const DefaultFailOn = SeverityMajor
