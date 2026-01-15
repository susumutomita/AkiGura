package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AuthToken holds the schema definition for the AuthToken entity.
type AuthToken struct {
	ent.Schema
}

// Fields of the AuthToken.
func (AuthToken) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable(),
		field.String("team_id").
			NotEmpty(),
		field.String("token").
			Unique().
			NotEmpty().
			Sensitive(),
		field.Time("expires_at"),
		field.Time("used_at").
			Optional().
			Nillable(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the AuthToken.
func (AuthToken) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("team", Team.Type).
			Ref("auth_tokens").
			Field("team_id").
			Unique().
			Required(),
	}
}

// Indexes of the AuthToken.
func (AuthToken) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("token").Unique(),
		index.Fields("team_id"),
	}
}
