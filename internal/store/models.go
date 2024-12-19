// General type definitions that any store implementation would use.
package store

type Decision struct {
	ActorUserID     string
	RecipientUserID string
	LikedRecipient  bool
	LastModified    int64
	SeenByRecipient bool
}

type DecisionFilter struct {
	ActorUserID     *string
	RecipientUserID *string
	LikedRecipient  *bool
	LastModified    *uint64
	SeenByRecipient *bool
}
