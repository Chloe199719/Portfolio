import type { PortableTextBlock } from "@portabletext/types";
export type ContentType =
  | "pageInfo"
  | "projects"
  | "note"
  | "photograph"
  | "nowUpdate"
  | "siteSettings";
export type ImageAsset = {
  _type: "image";
  asset: { _type: "reference"; _ref: string };
  alt?: string;
  caption?: string;
  hotspot?: { x: number; y: number; width: number; height: number };
};
export type RichBlock = PortableTextBlock | (ImageAsset & { _key: string });
export interface ContentDoc {
  _id: string;
  _type: ContentType;
  _rev?: string;
  _createdAt?: string;
  _updatedAt?: string;
  _publishedSlug?: string;
  title?: string;
  slug?: { _type?: "slug"; current: string };
  summary?: string;
  body?: RichBlock[];
  image?: ImageAsset;
  heroImage?: ImageAsset;
  profilePic?: ImageAsset;
  featured?: boolean;
  order?: number;
  date?: string;
  category?: string;
  tags?: string[];
  location?: string;
  linkToBuild?: string;
  projectRole?: string;
  problem?: string;
  decisions?: string;
  outcome?: string;
  technologies?: { _ref?: string; title?: string }[];
  name?: string;
  role?: string;
  email?: string;
  intro?: string;
  bio?: string;
  interests?: string[];
  homepageTitle?: string;
  homepageSubtitle?: string;
  modules?: {
    id: "work" | "photos" | "now" | "notes" | "guestbook";
    visible: boolean;
  }[];
}
export type Project = ContentDoc & {
  title: string;
  summary: string;
  slug: { current: string };
  technologyNames: string[];
};
export interface SiteContent {
  profile: ContentDoc;
  settings: ContentDoc;
  projects: Project[];
  notes: ContentDoc[];
  photos: ContentDoc[];
  updates: ContentDoc[];
  socials: { title: string; url: string }[];
}
export interface SessionUser {
  uid: string;
  name: string;
  owner: boolean;
}
export interface GuestEntry {
  id: string;
  name: string;
  message: string;
  createdAt: string;
  status: "pending" | "approved" | "rejected";
  own?: boolean;
}
