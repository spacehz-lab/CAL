package surface

import (
	"encoding/json"
	"strings"

	"github.com/spacehz-lab/cal/internal/llm"
)

const promptKeyAcquisitionHint = "acquisition_hint"

const systemPrompt = `Return only JSON. Extract documented CLI operation surfaces from observations.

Goal:
Stage 1 builds a bounded entry-point inventory for later Capability planning. It does not decide capabilities or prove behavior.

Response shape:
{"surface_items":[{"id":"s1","kind":"command|subcommand|mode|option","name":"...","usage":"optional documented invocation shape","description":"...","evidence_source":"help|man|stdout","decision":"keep|defer|skip","reason":"short decision reason"}]}.

Boundary:
- Do not choose capability_id, execution args, probe inputs, verify specs, verifiers, or pass/fail outcomes.
- Surface can only decide whether an observed CLI surface is worth considering later.
- Descriptions and reasons must be grounded in observations and must not infer broader reusable capability semantics.
- acquisition_hint is an optional natural-language description of what the caller wants to acquire. Use it only as a relevance hint when deciding which observed CLI surfaces to keep.
- Do not invent surfaces that are not documented in observations. Do not return an empty surface_items list solely because acquisition_hint is not an exact command name.
- usage is optional. If observations include a Usage line, command synopsis, or directly documented invocation shape for the surface, copy the closest provider-specific invocation shape into usage.
- Do not invent usage flags, arguments, paths, or modes that are not present in observations. Omit usage when no invocation shape is visible.

Internal decision process:
Before writing JSON, classify each candidate surface internally in this order:
1. Is it a documented CLI entry point in observations? If not, skip.
2. Is it only metadata, help, version, usage, self-documentation, or an alias without added operation coverage? If yes, skip.
3. Does the observed name or description provide a stable operation meaning suitable for Capability planning? If yes, keep.
4. Is it potentially useful but too shallow, ambiguous, interactive, server/listener, protocol-specific, low-level, or in need of command-specific help, man output, or safer inspection? If yes, defer.
5. Otherwise skip.
Do not output hidden reasoning steps. Output only the JSON object with one short reason per item.

Kind rules:
- Use kind only from command, subcommand, mode, option.
- Use command for primary commands and subcommand for documented nested commands.
- Use mode for documented operating modes.
- Use option when a CLI exposes useful behavior primarily through flags.
- For flag-driven CLIs, emit each documented operation flag as kind="option" instead of subcommand.

Keep rules:
- Use keep when the observation, command name, or option name gives a stable reusable operation meaning suitable for Capability planning.
- For command-list CLIs, include documented primary commands broadly up to max_surface_items.
- Prefer primary commands and command families over every algorithm, format, cipher, digest, flag variant, alias, or metadata-only entry.
- For complex command-list CLIs, do not keep every primary command.
- Keep commands with clear common operation names, clear data/object names, or explicit reusable-operation descriptions.
- Keep core state-changing operations such as install, update, upgrade, uninstall, link, unlink, pin, unpin, tap, untap, and cleanup when documented.
- Do not defer solely because a surface is state-changing, network-dependent, configuration-changing, destructive, or may require confirmation.
- Those risks belong to later Binding, Verification, or Use policy.

Defer rules:
- Use defer only when the current observation is too shallow or ambiguous to infer a stable semantic operation.
- Use defer when the surface is interactive, server/listener, protocol-specific, low-level, or clearly needs command-specific help, man output, or safer inspection.

Skip rules:
- Use skip for metadata-only, self-documentation, or alias-only entries.
- Skip individual algorithms, formats, ciphers, digests, flag variants, and metadata entries unless the CLI exposes useful behavior primarily through that option.`

func prompt(req *Request) *llm.Request {
	payload, _ := json.Marshal(map[string]any{
		"provider":               req.Provider,
		"observations":           req.Observations,
		promptKeyAcquisitionHint: strings.TrimSpace(req.Hint),
		"max_surface_items":      req.MaxItems,
	})
	return &llm.Request{System: systemPrompt, User: string(payload), JSON: true}
}
