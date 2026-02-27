package journal

import (
	"fmt"
	"os/exec"
	"strings"
)

// Sync synchronises the journal directory with the configured rclone remote.
// Only day-record files matching the YYYY-MM-DD.json pattern are transferred;
// exports/ and config.json are intentionally left device-local.
//
// Supported directions:
//
//	"push"  – copy local journal dir → remote
//	"pull"  – copy remote            → local journal dir
//	"both"  – pull first, then push (merge; both sides end up with all files)
//	""      – treated as "both"
func Sync(cfg SyncConfig) error {
	if cfg.Remote == "" {
		return fmt.Errorf("sync remote is not configured; set \"sync.remote\" in ~/.journal/config.json")
	}
	dir, err := Dir()
	if err != nil {
		return err
	}
	direction := cfg.Direction
	if direction == "" {
		direction = "both"
	}
	// Only transfer day-record files; leave everything else device-local.
	// Pattern ????-??-??.json matches YYYY-MM-DD.json exactly (? = one character).
	filter := []string{"--include", "????-??-??.json", "--exclude", "*"}
	switch direction {
	case "pull":
		return rcloneCopy(cfg.Remote, dir, filter)
	case "push":
		return rcloneCopy(dir, cfg.Remote, filter)
	case "both":
		if err := rcloneCopy(cfg.Remote, dir, filter); err != nil {
			return fmt.Errorf("pull: %w", err)
		}
		if err := rcloneCopy(dir, cfg.Remote, filter); err != nil {
			return fmt.Errorf("push: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unknown sync direction %q; use \"push\", \"pull\", or \"both\"", direction)
	}
}

// rcloneCopy runs: rclone copy <src> <dst> [extraArgs...]
func rcloneCopy(src, dst string, extraArgs []string) error {
	args := append([]string{"copy", src, dst}, extraArgs...)
	cmd := exec.Command("rclone", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}
