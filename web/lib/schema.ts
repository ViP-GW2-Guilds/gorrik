import {
  pgTable,
  text,
  uuid,
  bigint,
  boolean,
  timestamp,
  primaryKey,
  uniqueIndex,
} from "drizzle-orm/pg-core";
import { sql } from "drizzle-orm";

export const encounters = pgTable("encounters", {
  id: text("id").primaryKey(),
  name: text("name").notNull(),
  category: text("category").notNull(),
  subcategory: text("subcategory").notNull(),
});

export const logs = pgTable("logs", {
  id: uuid("id").primaryKey().default(sql`gen_random_uuid()`),
  filename: text("filename").notNull().unique(),
  encounterId: text("encounter_id").references(() => encounters.id),
  encounterName: text("encounter_name").notNull(),
  category: text("category").notNull(),
  subcategory: text("subcategory").notNull(),
  result: text("result").notNull(),
  mode: text("mode").notNull(),
  durationMs: bigint("duration_ms", { mode: "number" }).notNull(),
  loggedAt: timestamp("logged_at", { withTimezone: true }).notNull(),
  uploadedAt: timestamp("uploaded_at", { withTimezone: true })
    .notNull()
    .default(sql`now()`),
  fileUrl: text("file_url").notNull(),
  isFavorite: boolean("is_favorite").notNull().default(false),
  tags: text("tags").array().default(sql`'{}'`),
  dpsReportUrl: text("dps_report_url"),
});

export const accounts = pgTable("accounts", {
  id: uuid("id").primaryKey().default(sql`gen_random_uuid()`),
  accountName: text("account_name").notNull().unique(),
});

export const characters = pgTable(
  "characters",
  {
    id: uuid("id").primaryKey().default(sql`gen_random_uuid()`),
    accountId: uuid("account_id")
      .notNull()
      .references(() => accounts.id),
    characterName: text("character_name").notNull(),
  },
  (t) => [uniqueIndex("characters_account_character_idx").on(t.accountId, t.characterName)]
);

export const logPlayers = pgTable(
  "log_players",
  {
    logId: uuid("log_id")
      .notNull()
      .references(() => logs.id, { onDelete: "cascade" }),
    characterId: uuid("character_id")
      .notNull()
      .references(() => characters.id),
    profession: text("profession").notNull(),
    eliteSpec: text("elite_spec"),
  },
  (t) => [primaryKey({ columns: [t.logId, t.characterId] })]
);

export type Log = typeof logs.$inferSelect;
export type NewLog = typeof logs.$inferInsert;
