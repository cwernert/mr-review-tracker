# zap/

Reference copies of the Zapier-side code that produces the data backing `mr-review-tracker`.

**Nothing in this folder is compiled into the Go binary.** It lives here so that the producer (a Code-by-Zapier step running in Zapier's cloud) can be versioned alongside the consumer (`storage.go` in the repo root) — bumping the schema in one without the other is exactly the kind of bug we want git history to catch.

## Where the Zap lives

<https://zapier.com/editor/223720761/>

(Editor-mode link — only the Zap owner / invited collaborators can open it.)

## Files

| File                                       | Purpose                                                                          |
| ------------------------------------------ | -------------------------------------------------------------------------------- |
| [`produce-storage.js`](produce-storage.js) | The Code-by-Zapier step that POSTs the MR list to the Storage by Zapier channel. |

## Deploying a change

1. Edit `produce-storage.js` here, commit, push.
2. Open the Zap in Zapier: <https://zapier.com/editor/223720761/>.
3. Find the **Run Javascript** step that talks to Storage by Zapier.
4. Paste the new contents over the old.
5. Test the step (Zapier's "Test action" button) and confirm the returned `mrCount` and `payloadKb` look sane.
6. Re-publish the Zap.

## Gotcha: Zapier censors "secrets" in stored output

Zapier scans every string a step produces and rewrites any substring it recognises as one of your registered secrets (auth credentials, env vars, the value of a connected app's token, etc.) into a marker of the shape:

```
:censored:<original-length>:<stable-hash>:
```

For this Zap, `gitlab.com` happens to match a registered GitLab auth credential, so without intervention every `web_url` we POST to Storage arrives looking like:

```
https://:censored:10:134f110505:/zapier/team-integrations/parser/-/merge_requests/241
```

…which is, of course, not a clickable URL.

`produce-storage.js` un-censors the host before posting by string-replacing the marker with `gitlab.com`. The marker (`CENSORED_HOST`) and the real host (`GITLAB_HOST`) are both constants at the top of the file. If Zapier ever changes their hashing algorithm the marker will stop matching and you'll see censored URLs in Storage again — fix is to copy the new marker from the Storage payload into `CENSORED_HOST`.

## Schema version

The producer emits `v: <SCHEMA_VERSION>` at the top of every record. The consumer (`storage.go`) checks that against its own `supportedSchemaVersion` constant and shows an actionable error in the menu bar if they disagree:

> Error: storage schema v0 not supported — update the producer Zap (need v1)

When you change the shape, bump both numbers in the same commit. The full schema is documented in the top-level [README](../README.md#storage-schema).
