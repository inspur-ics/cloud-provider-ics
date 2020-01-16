package icslib

// FindFCD is the type that represents the types of searches used to
// discover FCDs.
type FindFCD int

const (
    // FindFCDByID finds FCDs with the provided ID.
    FindFCDByID FindFCD = iota // 0

    // FindFCDByName finds FCDs with the provided name.
    FindFCDByName // 1
)