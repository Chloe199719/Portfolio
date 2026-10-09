export const shapeNames = [
  "Circle",
  "Diamond",
  "Triangle",
  "Square",
  "Cross",
  "Rings",
  "Bars",
  "Arch",
];
export function shuffledDeck(random = Math.random) {
  const deck = Array.from({ length: 16 }, (_, i) => i % 8);
  for (let i = deck.length - 1; i > 0; i--) {
    const j = Math.floor(random() * (i + 1));
    [deck[i], deck[j]] = [deck[j], deck[i]];
  }
  return deck;
}
