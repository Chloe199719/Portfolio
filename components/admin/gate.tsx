"use client";
import { useSession } from "@/lib/client-api";
import { AuthControls } from "@/components/interactive/auth-controls";
import { PageHeading } from "@/components/site/ui";
import { Dashboard } from "./dashboard";
export function AdminGate() {
  const session = useSession();
  if (session.loading)
    return (
      <div className="shell py-16">
        <div className="skeleton h-40" aria-label="Checking your session" />
      </div>
    );
  if (session.user?.owner) return <Dashboard />;
  return (
    <div className="shell page-bottom admin">
      <PageHeading eyebrow="Owner access" title="Make this space yours.">
        <p>
          Sign in with your owner account and authenticator to publish and
          manage your website.
        </p>
      </PageHeading>
      <AuthControls
        user={session.user}
        configured={session.configured}
        onChange={session.refresh}
      />
      {session.user && (
        <p className="form-error" role="alert">
          This account does not have owner access. Sign in again with your
          additional factor.
        </p>
      )}
      {session.error && (
        <p className="form-error" role="alert">
          {session.error}
        </p>
      )}
    </div>
  );
}
