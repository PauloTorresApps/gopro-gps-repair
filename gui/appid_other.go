//go:build !linux

package main

// setLinuxAppID só tem efeito no Linux.
func setLinuxAppID() {}
