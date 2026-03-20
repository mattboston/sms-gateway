import { useTheme, type ThemeMode } from '@/lib/theme';

const options: Array<{ value: ThemeMode; label: string }> = [
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
  { value: 'system', label: 'System' },
];

export default function ThemeModeControl() {
  const { themeMode, effectiveTheme, setThemeMode } = useTheme();

  return (
    <label className="inline-flex items-center gap-2 text-sm text-gray-600 dark:text-[#93a1a1]">
      <span className="sr-only">Theme mode</span>
      <span aria-hidden="true" className="text-base leading-none">
        {effectiveTheme === 'dark' ? '🌙' : '☀️'}
      </span>
      <select
        aria-label="Theme mode"
        value={themeMode}
        onChange={(e) => setThemeMode(e.target.value as ThemeMode)}
        className="rounded-md border border-gray-300 bg-white px-2 py-1 text-sm text-gray-700 outline-none transition-colors hover:bg-gray-50 focus:ring-2 focus:ring-blue-500 dark:border-[#586e75] dark:bg-[#073642] dark:text-[#93a1a1] dark:hover:bg-[#0a4452] dark:focus:ring-[#268bd2]"
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  );
}
