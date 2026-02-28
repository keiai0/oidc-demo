CREATE SCHEMA IF NOT EXISTS "rp";
--> statement-breakpoint
CREATE TABLE "rp"."sessions" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"user_id" uuid NOT NULL,
	"op_session_id" varchar(255),
	"access_token" text NOT NULL,
	"refresh_token" text,
	"id_token" text NOT NULL,
	"token_expires_at" timestamp with time zone NOT NULL,
	"expires_at" timestamp with time zone NOT NULL,
	"revoked_at" timestamp with time zone,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "rp"."users" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"op_sub" varchar(255) NOT NULL,
	"email" varchar(255),
	"name" varchar(255),
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"updated_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "users_op_sub_unique" UNIQUE("op_sub")
);
--> statement-breakpoint
ALTER TABLE "rp"."sessions" ADD CONSTRAINT "sessions_user_id_users_id_fk" FOREIGN KEY ("user_id") REFERENCES "rp"."users"("id") ON DELETE no action ON UPDATE no action;