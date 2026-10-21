package sim

// autoCastTargetAdmitted is common to automatic target selection regardless
// of spell or support setting (TRIG-OFFMAP-041, AI-LIST-003).
func autoCastTargetAdmitted(target Entity) bool { return !target.OffMap }
