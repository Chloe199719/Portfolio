import Link from "next/link";
import { ArrowTopRightIcon } from "@radix-ui/react-icons";
import { getContent } from "@/lib/content";
export async function Footer() {
  const { socials } = await getContent();
  return (
    <footer className="site-footer">
      <div className="shell">
        <div className="footer-top">
          <div>
            <p className="eyebrow">Good things start with a hello.</p>
            <Link href="/contact" className="footer-title">
              Let’s make a connection.
              <ArrowTopRightIcon />
            </Link>
          </div>
          <Link className="wordmark" href="/">
            chloe<span>.</span>
          </Link>
        </div>
        <div className="footer-bottom">
          <span>© {new Date().getFullYear()} Chloe Pratas</span>
          <nav aria-label="Social links" className="flex flex-wrap gap-6">
            {socials.map((s) => (
              <a
                key={s.url}
                href={s.url}
                target="_blank"
                rel="noopener noreferrer"
              >
                {s.title}
              </a>
            ))}
          </nav>
          <div className="flex gap-5">
            <Link href="/privacy">Privacy</Link>
            <Link href="/admin">Owner sign-in</Link>
          </div>
        </div>
      </div>
    </footer>
  );
}
