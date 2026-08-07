/** Arabic / Persian (Farsi) / Hebrew / Syriac blocks used for RTL detection. */
const RTL_CHAR_RE =
  /[֐-׿؀-ۿݐ-ݿࢠ-ࣿיִ-﷿ﹰ-﻿]/;

/** Returns true when text contains a strong RTL script character. */
export function hasRTLText(text: string | null | undefined): boolean {
  if (!text) return false;
  return RTL_CHAR_RE.test(text);
}

/** Prefer browser auto direction; fall back to rtl when Arabic/Persian is present. */
export function textDirection(text: string | null | undefined): 'rtl' | 'ltr' | 'auto' {
  if (hasRTLText(text)) return 'rtl';
  return 'auto';
}
