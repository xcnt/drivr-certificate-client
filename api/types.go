package api

import "github.com/google/uuid"

type UUID string
type Timespan string

func NewGraphQLUUID(uuid uuid.UUID) UUID {
	return UUID(uuid.String())
}

func NewTimespan(timespan string) Timespan {
	return Timespan(timespan)
}
