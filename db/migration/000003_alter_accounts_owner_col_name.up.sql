ALTER TABLE "accounts" RENAME COLUMN "owner" TO "user_id";

ALTER TABLE "accounts" ALTER COLUMN "user_id" TYPE bigint USING "user_id"::bigint;

ALTER TABLE "accounts" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id");
