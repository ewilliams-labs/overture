# ADR 003: LLM Selection for Intent Analysis

**Status:** Proposed  
**Deciders:** Erik Williams (Principal Engineer)  
**Date:** 2026-02-12  

## Context
Overture requires a local LLM to act as the "Intent Compiler." This engine must translate abstract, multi-layered natural language requests (e.g., *"Make it sound like a rainy day in Seattle, but with a Willie Nelson vibe"*) into a structured JSON object. This requires both high-level music theory reasoning and strict adherence to a specific JSON schema.

## Options Considered

| Feature | **DeepSeek-R1 (8B Distilled)** | **Llama 3.1 (8B)** | **Mistral (7B v0.3)** |
| :--- | :--- | :--- | :--- |
| **Primary Logic** | **Reasoning (CoT)** | Instruction Following | General Conversational |
| **JSON Handling** | High (via Ollama `format:json`) | High (Native) | Moderate |
| **Musical Nuance** | **Superior (Deep Reasoning)** | Good (Pattern Match) | Standard |
| **Negative Constraints** | **High Precision** | Occasional Hallucination | Moderate |

## Pros & Cons

### 1. DeepSeek-R1 (Selected)
* **Pros:** The "Chain of Thought" (CoT) capability allows the model to "think through" the musical implications of a request. For example, it can logically deduce that "no auto-tune" should map to `acousticness > 0.7`.
* **Pros:** Extremely cost-efficient and lightweight for its reasoning depth.
* **Cons:** Can be "chatty" or slow due to the internal reasoning steps (mitigated by Ollama's structured output settings).

### 2. Llama 3.1
* **Pros:** The industry standard for small-model instruction following. Extremely reliable for simple JSON tasks.
* **Cons:** Lacks the "Reasoning" trace. It tends to match keywords rather than understanding the *intent* behind abstract descriptions like "Seattle rainy day."

### 3. Mistral
* **Pros:** Very fast inference.
* **Cons:** In our early testing, it struggled with complex nested JSON objects compared to the Llama-based distillations.

## Decision
We will use **DeepSeek-R1 (8B Distilled Llama)** via Ollama.

## Rationale
Overture’s core value is its ability to understand **vibe subtext**. DeepSeek-R1’s ability to reason through constraints before outputting data ensures that "vibe-based" filtering is accurate and musically sound. By using the distilled 8B version, we maintain low latency for a local-first experience on consumer hardware (Surface Pro).

## Consequences
* **Inference Speed:** There may be a slight "thinking delay" while the model processes the CoT.
* **Schema Requirement:** We must strictly enforce the `IntentObject` schema in our system prompt to ensure the output remains compatible with the Go backend.
