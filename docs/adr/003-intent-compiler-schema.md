# ADR 003: Intent Compiler & Multi-Intent Schema

## IntentObject Schema (Go)

```go
type VibeConstraint struct {
 Target float64 `json:"target,omitempty"`
 Min    float64 `json:"min,omitempty"`
 Max    float64 `json:"max,omitempty"`
 Weight string  `json:"weight"` // "REQUIRED", "HIGH", "LOW"
}

type IntentObject struct {
 IntentType string `json:"intent_type"` // CREATE, MODIFY, REORDER
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
  Pattern     string `json:"pattern"` // ARC, LINEAR, RANDOM
  Description string `json:"description"`
 } `json:"sequence"`
 Explanation string `json:"explanation"`
}
```

## System Prompt (AI Logic)

```text
You are the Overture Music Intent Engine. Your goal is to translate abstract human desires into a structured JSON 'IntentObject'.

Rules:
Reasoning: Use your internal logic to map stylistic requests (e.g., 'no auto-tune') to technical constraints (e.g., 'acousticness.min: 0.8').
Entities: Extract specific artists or genres mentioned.
Output: Return ONLY a valid JSON object. No conversational text.
Vibe Scaling: Energy and Valence are 0.0 to 1.0.
Example Mapping: 'I want a sad acoustic set' -> { 'vibe_constraints': { 'valence': {'target': 0.2}, 'acousticness': {'min': 0.7} } }
```
