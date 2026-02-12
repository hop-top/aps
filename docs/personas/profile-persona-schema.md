# Profile Persona Schema

## Description

The `persona` configuration object within a profile's `profile.yaml` that shapes AI behavior for the agent identity. Unlike actor personas (User, Maintainer, External Client), this is a data model concept that configures how a profile's agent presents itself.

## Schema

Defined in the profile data model (`specs/001-build-cli-core/data-model.md`):

```yaml
persona:
  tone: string    # Communication style (e.g., "formal", "casual", "technical")
  style: string   # Response format preference (e.g., "concise", "detailed", "structured")
  risk: string    # Risk tolerance level (e.g., "conservative", "moderate", "aggressive")
```

## Location

Stored at `~/.agents/profiles/<id>/profile.yaml` under the `persona` key.

## Usage

The persona configuration is optional. When present, it provides behavioral hints that can be consumed by AI agents operating within the profile's context. The values are free-form strings, allowing profiles to express nuanced behavioral preferences without a rigid enum.

## Example

```yaml
id: security-auditor
display_name: Security Auditor

persona:
  tone: formal
  style: structured
  risk: conservative
```
