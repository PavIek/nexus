//go:build !linux

package main

func pinCurrentThread(cpu int) error {
	return nil
}
