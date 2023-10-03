ALTER TABLE IF EXISTS "accounts" DROP CONSTRAINT IF EXISTS "accounts_user_id_fkey";

ALTER TABLE IF EXISTS "accounts" ALTER COLUMN "user_id" TYPE varchar USING "user_id"::varchar;

ALTER TABLE IF EXISTS "accounts" RENAME COLUMN "user_id" TO "owner";
