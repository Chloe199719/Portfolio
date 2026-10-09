"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import {
  ArrowTopRightIcon,
  ChevronDownIcon,
  Cross1Icon,
  HamburgerMenuIcon,
} from "@radix-ui/react-icons";
const main = [
  ["/work", "Work"],
  ["/notes", "Notes"],
  ["/photography", "Photos"],
  ["/about", "About"],
];
const extra = [
  ["/now", "Now"],
  ["/playground", "Playground"],
  ["/guestbook", "Guestbook"],
];
export function Header() {
  const pathname = usePathname();
  const [open, setOpen] = useState(false);
  const [more, setMore] = useState(false);
  const ref = useRef<HTMLElement>(null);
  useEffect(() => {
    function close(e: PointerEvent) {
      if (!ref.current?.contains(e.target as Node)) {
        setMore(false);
        setOpen(false);
      }
    }
    function escape(e: KeyboardEvent) {
      if (e.key === "Escape") {
        setMore(false);
        setOpen(false);
      }
    }
    document.addEventListener("pointerdown", close);
    document.addEventListener("keydown", escape);
    return () => {
      document.removeEventListener("pointerdown", close);
      document.removeEventListener("keydown", escape);
    };
  }, []);
  const link = (href: string, label: string) => (
    <Link
      key={href}
      href={href}
      aria-current={
        pathname === href || pathname?.startsWith(href + "/")
          ? "page"
          : undefined
      }
      onClick={() => {
        setOpen(false);
        setMore(false);
      }}
    >
      {label}
    </Link>
  );
  return (
    <header ref={ref} className="site-header">
      <div className="shell flex items-center justify-between gap-6 py-6">
        <Link className="wordmark" href="/" aria-label="Chloe, home">
          chloe<span>.</span>
        </Link>
        <nav aria-label="Main navigation" className="desktop-nav">
          {main.map(([href, label]) => link(href, label))}
          <div className="relative">
            <button
              className="nav-more"
              aria-expanded={more}
              aria-controls="more-links"
              onClick={() => setMore(!more)}
            >
              More <ChevronDownIcon />
            </button>
            {more && (
              <div id="more-links" className="nav-popover">
                {extra.map(([href, label]) => link(href, label))}
              </div>
            )}
          </div>
        </nav>
        <Link href="/contact" className="header-contact">
          Let’s talk <ArrowTopRightIcon />
        </Link>
        <button
          className="mobile-toggle icon-button"
          aria-label={open ? "Close navigation" : "Open navigation"}
          aria-expanded={open}
          aria-controls="mobile-nav"
          onClick={() => setOpen(!open)}
        >
          {open ? <Cross1Icon /> : <HamburgerMenuIcon />}
        </button>
      </div>
      {open && (
        <nav
          id="mobile-nav"
          aria-label="Mobile navigation"
          className="mobile-nav shell"
        >
          {[...main, ...extra, ["/contact", "Contact"]].map(([href, label]) =>
            link(href, label),
          )}
        </nav>
      )}
    </header>
  );
}
