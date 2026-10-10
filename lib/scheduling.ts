// Interpret the owner's wall-clock input in Berlin, including its UTC offset.
export function berlinToUTC(value: string) {
  const guess = new Date(`${value}:00Z`);
  if (Number.isNaN(+guess)) throw new Error("Choose a date and time.");
  const parts = (d: Date) =>
    Object.fromEntries(
      new Intl.DateTimeFormat("en-CA", {
        timeZone: "Europe/Berlin",
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
        hourCycle: "h23",
      })
        .formatToParts(d)
        .map((p) => [p.type, p.value]),
    );
  let result = guess;
  for (let i = 0; i < 3; i++) {
    const p = parts(result);
    const local = Date.parse(
      `${p.year}-${p.month}-${p.day}T${p.hour}:${p.minute}:00Z`,
    );
    result = new Date(+result + (+guess - local));
  }
  const p = parts(result);
  if (`${p.year}-${p.month}-${p.day}T${p.hour}:${p.minute}` !== value)
    throw new Error(
      "That time does not exist during the Berlin daylight-saving change. Choose another time.",
    );
  return result.toISOString();
}
