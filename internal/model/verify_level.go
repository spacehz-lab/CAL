package model

// VerifyLevelRank returns a comparable rank for verification levels.
func VerifyLevelRank(level VerifyLevel) int {
	switch level {
	case VerifyLevelL1:
		return 1
	case VerifyLevelL2:
		return 2
	case VerifyLevelL3:
		return 3
	default:
		return 0
	}
}
