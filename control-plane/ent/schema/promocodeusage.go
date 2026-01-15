package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PromoCodeUsage holds the schema definition for the PromoCodeUsage entity.
type PromoCodeUsage struct {
	ent.Schema
}

// Fields of the PromoCodeUsage.
func (PromoCodeUsage) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable(),
		field.Time("applied_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the PromoCodeUsage.
func (PromoCodeUsage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("promo_code", PromoCode.Type).
			Ref("usages").
			Unique().
			Required(),
		edge.From("team", Team.Type).
			Ref("promo_code_usages").
			Unique().
			Required(),
	}
}

// Indexes of the PromoCodeUsage.
func (PromoCodeUsage) Indexes() []ent.Index {
	return []ent.Index{
		index.Edges("team"),
		index.Edges("promo_code", "team").Unique(),
	}
}
