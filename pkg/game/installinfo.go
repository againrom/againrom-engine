package game

import (
	"againrom/pkg/base"
	"againrom/pkg/installinfo"
)

// InstallInfo is what a cheap look at a directory says about it as an install.
type InstallInfo = installinfo.Info

// DetectBase forwards to the one install detection.
func DetectBase(root string) (base.Match, error) { return installinfo.Detect(root) }

// InspectInstall forwards to the one install inspection.
func InspectInstall(root string) InstallInfo { return installinfo.Inspect(root) }
