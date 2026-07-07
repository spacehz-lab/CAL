package capability

import (
	"encoding/json"
	"strings"

	"github.com/spacehz-lab/cal/internal/llm"
)

const promptKeyAcquisitionHint = "acquisition_hint"

const systemPrompt = `Return only JSON. Map kept CLI surfaces to provider-independent capability ids.

Goal:
Stage 2 is semantic planning. It names the reusable operation that later Binding and Evidence stages may implement and verify.

Response shape:
{"capabilities":[{"capability_id":"subject.operation","description":"...","source_surface_ids":["s1"],"confidence":"high|medium|low"}]}.

Boundary:
- Do not produce execution args, probes, verifiers, verify specs, pass/fail claims, or input schemas.
- Do not decide whether the provider-specific execution works.
- Do not output hidden reasoning steps. Output only the JSON object.
- acquisition_hint is an optional natural-language description of what the caller wants to acquire. Use it only to prefer relevant capability plans that are supported by supplied surface_items.
- If acquisition_hint is present, treat it as a narrowing constraint. Return only the smallest set of capability plans directly needed to satisfy the hinted task. Exclude merely adjacent, opposite, follow-up, maintenance, diagnostic, or broadly related capabilities. If acquisition_hint is absent, keep broad discovery.

Internal decision process:
Before writing JSON, decide each capability internally in this order:
1. Identify the semantic subject of the kept surface without using provider, executable, command, flag, format, algorithm, mode, or artifact wording.
2. Identify the semantic operation direction. Opposite operations must stay separate.
3. Reuse an existing_capabilities id only when both its id and description are valid and semantically equivalent.
4. Use capability_policy.preferred_subjects and capability_policy.preferred_operations when they express the operation accurately.
5. Create a new subject or operation only when the supplied vocabulary cannot express the observed operation.
6. Merge source surfaces only when they describe the same subject and operation and can plausibly share compatible Binding inputs, execution shape, and output semantics.
7. Split or keep only the clearest directly executable source when surfaces differ by command family, required inputs, execution shape, output semantics, or operation direction.

Capability id rules:
- capability_id must be exactly two lowercase dotted parts: <subject>.<operation>.
- Choose subject from capability_policy.preferred_subjects whenever a semantically correct subject exists.
- Choose operation from capability_policy.preferred_operations whenever a semantically correct operation exists.
- Do not create protocol-specific, format-specific, algorithm-specific, or implementation-specific subjects when a supplied generic subject preserves the reusable operation meaning.
- Create a new lowercase subject or operation only when observations cannot be expressed by supplied generic terms.
- New terms must be minimal, generic, reusable, and provider-independent.
- capability_id must describe the reusable semantic operation, not the provider surface.
- Do not include provider names, executable names, command names, flags, paths, versions, formats, encodings, algorithms, modes, variants, target artifact kinds, or input/output media in capability_id.

Discriminator rules:
- Runtime-selectable or binding-specific details belong to later Binding execution inputs and Evidence checks, not capability_id.
- If the same reusable operation can vary by format, encoding, algorithm, mode, variant, artifact kind, or concrete I/O shape, keep the capability_id generic.
- Do not maintain a format-by-format or algorithm-by-algorithm whitelist. Apply the same discriminator rule to new future variants.

Grouping rules:
- Merge surfaces only when they share the same semantic subject and operation.
- Merged surfaces must plausibly share compatible Binding inputs, execution shape, and output semantics.
- Do not merge surfaces that require clearly different inputs, execution shapes, output semantics, command families, or opposite operations.
- Prefer the most direct source surface for each capability.
- Do not add extra source_surface_ids merely because another command can also participate in the broad operation.
- If the same semantic operation appears across different command families, prefer separate narrower capabilities or keep only the clearest directly executable source surface.
- source_surface_ids must only reference ids from the supplied surface_items.

Reuse and description rules:
- Reuse an existing_capabilities id only when its id and description follow these rules and are semantically equivalent.
- Create a new capability when no existing capability clearly describes the operation.
- Do not reuse invalid, provider-specific, command-specific, or discriminator-specific existing ids.
- description must be provider-independent and must describe exactly the selected semantic capability.
- description must not add unsupported operations, hide opposite operations, or narrow capability_id with a format, encoding, algorithm, mode, or target artifact discriminator.
- confidence="high" only when the kept surface name and description clearly support the semantic capability.
- Use confidence="medium" when the operation is supported but the subject, direction, or grouping is partly inferred.
- Use confidence="low" when the mapping is weak but still worth exploring; omit the capability if it is not supported by the supplied surface_items.
- If acquisition_hint is present, prefer capability plans relevant to it, but do not invent unsupported capabilities and do not require an exact capability_id match.`

func prompt(req *Request) *llm.Request {
	payload, _ := json.Marshal(map[string]any{
		"provider":               req.Provider,
		"surface_items":          req.Surfaces,
		"existing_capabilities":  req.Catalog,
		"capability_policy":      req.Policy,
		promptKeyAcquisitionHint: strings.TrimSpace(req.Hint),
		"max_capabilities":       req.MaxPlans,
	})
	return &llm.Request{System: systemPrompt, User: string(payload), JSON: true}
}
