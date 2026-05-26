/**
 * produce-storage.js — reference copy of the Code-by-Zapier step that
 * publishes the data backing mr-review-tracker.
 *
 * This file is NOT compiled into the Go binary; it is checked in so the
 * producer schema is versioned alongside the consumer (storage.go).
 *
 * To deploy: paste the contents of this file into the "Run Javascript"
 * step of the Zap that posts to Storage by Zapier, then re-publish.
 *
 * Expected inputData:
 *   gitlab    — JSON string of an array of GitLab MergeRequest objects
 *                (as returned by the GitLab "List merge requests" endpoint).
 *   timestamp — ISO-8601 string of when this Zap run started.
 *
 * Output schema (must match `supportedSchemaVersion` in storage.go):
 *   {
 *     "v": 1,
 *     "fetched_at": "2026-05-26T00:51:43+00:00",
 *     "mrs": [
 *       {
 *         "title":      string,
 *         "url":        string,
 *         "project":    "group/project!123",
 *         "author":     string (username),
 *         "author_url": string,
 *         "updated_at": ISO-8601 string,
 *         "created_at": ISO-8601 string,
 *         "labels":     string[]  // capped at 10
 *         "draft":      boolean
 *       },
 *       ...
 *     ]
 *   }
 *
 * Bump SCHEMA_VERSION here in lock-step with supportedSchemaVersion in
 * storage.go whenever the shape changes.
 */

const SCHEMA_VERSION = 1;
const CHANNEL_SECRET = "fbdee107-eb6b-47d5-ba86-0aaa8a41b813";
const STORAGE_URL = "https://store.zapier.com/api/records";
const MAX_LABELS_PER_MR = 10;

const { gitlab = "[]", timestamp } = inputData;

let mrs;
try {
  mrs = JSON.parse(gitlab);
} catch (err) {
  throw new Error(`Could not parse \`gitlab\` input as JSON: ${err.message}`);
}
if (!Array.isArray(mrs)) {
  throw new Error("`gitlab` input did not parse to an array");
}

// Whitelisted shape. Any field the Go app does not read is dropped here so
// Storage by Zapier never balloons with payload we'll never use.
const slim = mrs.map((mr) => ({
  title: String(mr.title ?? ""),
  url: String(mr.web_url ?? ""),
  project: String((mr.references && mr.references.full) ?? ""),
  author: String((mr.author && mr.author.username) ?? ""),
  author_url: String((mr.author && mr.author.web_url) ?? ""),
  updated_at: mr.updated_at ?? null,
  created_at: mr.created_at ?? null,
  labels: Array.isArray(mr.labels) ? mr.labels.slice(0, MAX_LABELS_PER_MR) : [],
  draft: Boolean(mr.draft ?? mr.work_in_progress),
}));

const payload = {
  v: SCHEMA_VERSION,
  fetched_at: timestamp ?? new Date().toISOString(),
  mrs: slim,
};

const body = JSON.stringify(payload);

const res = await fetch(STORAGE_URL, {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    Accept: "application/json",
    "X-SECRET": CHANNEL_SECRET,
  },
  body,
});

const result = await res.json();

return {
  result,
  mrCount: slim.length,
  payloadBytes: body.length,
  payloadKb: (body.length / 1024).toFixed(2),
};
