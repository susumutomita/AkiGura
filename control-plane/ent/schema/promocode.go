package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PromoCode holds the schema definition for the PromoCode entity.
type PromoCode struct {
	ent.Schema
}

// Fields of the PromoCode.
func (PromoCode) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").
			Unique().
			Immutable(),
		field.String("code").
			Unique().
			NotEmpty(),
		field.Enum("discount_type").
			Values("percent", "fixed"),
		field.Int("discount_value").
			Positive(),
		field.String("applies_to").
			Optional().
			Nillable(), // NULL = all plans
		field.Time("valid_from").
			Default(time.Now),
		field.Time("valid_until").
			Optional().
			Nillable(),
		field.Int("max_uses").
			Optional().
			Nillable(),
		field.Int("uses_count").
			Default(0),
		field.Bool("enabled").
			Default(true),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

// Edges of the PromoCode.
func (PromoCode) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("usages", PromoCodeUsage.Type),
	}
}

// Indexes of the PromoCode.
func (PromoCode) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code").Unique(),
	}
}
