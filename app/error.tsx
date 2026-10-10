"use client";
export default function ErrorPage({ reset }: { reset: () => void }) {
  return (
    <div className="shell page-bottom">
      <div className="page-heading">
        <p className="eyebrow">A brief interruption</p>
        <h1>Something didn’t load.</h1>
        <p className="page-intro">Please try again in a moment.</p>
      </div>
      <button className="button" onClick={reset}>
        Try again
      </button>
    </div>
  );
}
