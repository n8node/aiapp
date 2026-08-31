const LOWER = "abcdefghijklmnopqrstuvwxyz";
const UPPER = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";
const DIGITS = "0123456789";
const SPECIAL = "!@#$%^&*-_=+";

function randomIndex(max: number): number {
  const buf = new Uint32Array(1);
  crypto.getRandomValues(buf);
  return buf[0] % max;
}

function pick(chars: string): string {
  return chars[randomIndex(chars.length)];
}

function shuffle(values: string[]): string[] {
  const arr = [...values];
  for (let i = arr.length - 1; i > 0; i--) {
    const j = randomIndex(i + 1);
    [arr[i], arr[j]] = [arr[j], arr[i]];
  }
  return arr;
}

export function generateSecurePassword(length = 16): string {
  const required = [pick(LOWER), pick(UPPER), pick(DIGITS), pick(SPECIAL)];
  const all = LOWER + UPPER + DIGITS + SPECIAL;
  while (required.length < length) {
    required.push(pick(all));
  }
  return shuffle(required).join("");
}

export async function copyToClipboard(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}
