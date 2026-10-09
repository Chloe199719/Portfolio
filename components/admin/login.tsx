"use client";
import { useRouter } from "next/navigation";
import { useSession } from "@/lib/client-api";
import { AuthControls } from "@/components/interactive/auth-controls";
export function AdminLogin() {
  const session = useSession();
  const router = useRouter();
  return (
    <div className="max-w-xl">
      <AuthControls
        user={session.user}
        configured={session.configured}
        onChange={async () => {
          await session.refresh();
          router.refresh();
        }}
        onComplete={() => router.refresh()}
      />
      {session.user && !session.user.owner && (
        <p role="alert" className="form-error">
          This account is not the configured site owner.
        </p>
      )}
      {session.error && (
        <p role="alert" className="form-error">
          {session.error}
        </p>
      )}
    </div>
  );
}
