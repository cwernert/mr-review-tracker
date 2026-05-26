# zap/

Reference copies of the Zapier-side code that produces the data backing `mr-review-tracker`.

**Nothing in this folder is compiled into the Go binary.** It lives here so that the producer (a Code-by-Zapier step running in Zapier's cloud) can be versioned alongside the consumer (`storage.go` in the repo root) — bumping the schema in one without the other is exactly the kind of bug we want git history to catch.

## Files

| File | Purpose |
|---|---|
| [`produce-storage.js`](produce-storage.js) | The Code-by-Zapier step that POSTs the MR list to the Storage by Zapier channel. |

## Deploying a change

1. Edit `produce-storage.js` here, commit, push.
2. Open the matching Zap in Zapier.
3. Find the **Run Javascript** step that talks to Storage by Zapier.
4. Paste the new contents over the old.
5. Test the step (Zapier's "Test action" button) and confirm the returned `mrCount` and `payloadKb` look sane.
6. Re-publish the Zap.

## Schema version

The producer emits `v: <SCHEMA_VERSION>` at the top of every record. The consumer (`storage.go`) checks that against its own `supportedSchemaVersion` constant and shows an actionable error in the menu bar if they disagree:

> Error: storage schema v0 not supported — update the producer Zap (need v1)

When you change the shape, bump both numbers in the same commit. The full schema is documented in the top-level [README](../README.md#storage-schema).
