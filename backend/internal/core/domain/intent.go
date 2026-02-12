package domain

type VibeConstraint struct {
	Target float64 `json:"target,omitempty"`
	Min    float64 `json:"min,omitempty"`
	Max    float64 `json:"max,omitempty"`
	Weight string  `json:"weight"`
}

type IntentObject struct {
	IntentType string `json:"intent_type"`
	Entities   struct {
		Artists []string `json:"artists"`
		Genres  []string `json:"genres"`
	} `json:"entities"`
	VibeConstraints struct {
		Energy     *VibeConstraint `json:"energy,omitempty"`
		Valence    *VibeConstraint `json:"valence,omitempty"`
		Acoustic   *VibeConstraint `json:"acousticness,omitempty"`
		Instrument *VibeConstraint `json:"instrumentalness,omitempty"`
	} `json:"vibe_constraints"`
	Sequence struct {
		Pattern     string `json:"pattern"`
		Description string `json:"description"`
	} `json:"sequence"`
	Explanation string `json:"explanation"`
}
