ALTER TABLE "accounts" DROP CONSTRAINT "accounts_user_id_fkey";

ALTER TABLE "accounts" ALTER COLUMN "user_id" TYPE varchar USING "user_id"::varchar;

ALTER TABLE "accounts" RENAME COLUMN "user_id" TO "owner";
