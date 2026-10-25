//go:build !windows && !darwin && !linux

package pathdialog

func show(request Request) (Result, error) {
	return unavailable(request)
}
