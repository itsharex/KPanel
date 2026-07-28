//go:build !linux

package webenv

import "os"

func ownerTrusted(os.FileInfo) bool { return false }

func currentEUID() int { return -1 }
