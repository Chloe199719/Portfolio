"use client";
import { useEffect, useRef, useState } from "react";
import { ReloadIcon } from "@radix-ui/react-icons";
import { shapeNames, shuffledDeck } from "@/lib/game";
export function MemoryGame() {
  const [deck, setDeck] = useState<number[]>([]);
  const [flipped, setFlipped] = useState<number[]>([]);
  const [matched, setMatched] = useState<number[]>([]);
  const [moves, setMoves] = useState(0);
  const [best, setBest] = useState<number | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const win = matched.length === 8;
  useEffect(() => {
    try {
      const saved = Number(localStorage.getItem("chloe-memory-best"));
      if (saved >= 8) setBest(saved);
    } catch {}
    return () => {
      if (timer.current) clearTimeout(timer.current);
    };
  }, []);
  function start() {
    if (timer.current) clearTimeout(timer.current);
    setDeck(shuffledDeck());
    setFlipped([]);
    setMatched([]);
    setMoves(0);
  }
  function flip(index: number) {
    if (
      !deck.length ||
      flipped.length === 2 ||
      flipped.includes(index) ||
      matched.includes(deck[index])
    )
      return;
    const next = [...flipped, index];
    setFlipped(next);
    if (next.length === 2) {
      const count = moves + 1;
      setMoves(count);
      if (deck[next[0]] === deck[next[1]]) {
        const pairs = [...matched, deck[index]];
        setMatched(pairs);
        setFlipped([]);
        if (pairs.length === 8 && (best === null || count < best)) {
          setBest(count);
          try {
            localStorage.setItem("chloe-memory-best", String(count));
          } catch {}
        }
      } else timer.current = setTimeout(() => setFlipped([]), 950);
    }
  }
  return (
    <div className="memory-game">
      <div className="game-toolbar">
        <div>
          <span className="eyebrow">Moves</span>
          <strong>{String(moves).padStart(2, "0")}</strong>
        </div>
        <div>
          <span className="eyebrow">Pairs</span>
          <strong>
            {matched.length}
            <span className="text-muted"> / 8</span>
          </strong>
        </div>
        <div>
          <span className="eyebrow">Personal best</span>
          <strong>{best ?? "—"}</strong>
        </div>
        <button
          className="icon-button"
          aria-label="Restart game"
          onClick={start}
        >
          <ReloadIcon />
        </button>
      </div>
      <div className="memory-grid" aria-label="Memory matching board">
        {Array.from({ length: 16 }, (_, index) => {
          const shape = deck[index];
          const visible = flipped.includes(index) || matched.includes(shape);
          return (
            <button
              key={index}
              className={`memory-tile ${visible ? "revealed" : ""} ${matched.includes(shape) ? "matched" : ""}`}
              disabled={!deck.length || matched.includes(shape)}
              onClick={() => flip(index)}
              aria-label={
                visible
                  ? `${shapeNames[shape]}, ${matched.includes(shape) ? "matched" : "revealed"}`
                  : `Reveal tile ${index + 1}`
              }
              aria-pressed={visible}
            >
              {visible ? (
                <span
                  className={`tile-shape shape-${shape}`}
                  aria-hidden="true"
                />
              ) : (
                <span className="tile-back" aria-hidden="true">
                  c.
                </span>
              )}
            </button>
          );
        })}
      </div>
      <div className="game-status" role="status" aria-live="polite">
        {!deck.length ? (
          <>
            <p>Sixteen tiles. Eight pairs. A small moment of focus.</p>
            <button className="button" onClick={start}>
              Start a round
            </button>
          </>
        ) : win ? (
          <>
            <p>All pairs found in {moves} moves. Nicely noticed.</p>
            <button className="button" onClick={start}>
              Play again
            </button>
          </>
        ) : (
          <p>
            {flipped.length === 2
              ? "Not a pair. Take another look."
              : `${matched.length} of 8 pairs found. Choose ${flipped.length ? "one more tile" : "two tiles"}.`}
          </p>
        )}
      </div>
    </div>
  );
}
