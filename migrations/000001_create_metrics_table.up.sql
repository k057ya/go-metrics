CREATE TABLE "public"."metrics" (
    "id" varchar NOT NULL,
    "type" varchar,
    "delta" int8,
    "value" double precision,
    "updated_at" timestamptz DEFAULT NOW(),
    PRIMARY KEY ("id")
);