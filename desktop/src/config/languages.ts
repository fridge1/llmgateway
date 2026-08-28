export interface Language {
  code: string;
  name: string;
  englishName: string;
}

export const languages: Language[] = [
  { code: "auto", name: "自动检测", englishName: "auto-detected language" },
  { code: "zh", name: "中文", englishName: "Chinese" },
  { code: "en", name: "英语", englishName: "English" },
  { code: "ja", name: "日语", englishName: "Japanese" },
  { code: "ko", name: "韩语", englishName: "Korean" },
  { code: "fr", name: "法语", englishName: "French" },
  { code: "de", name: "德语", englishName: "German" },
  { code: "es", name: "西班牙语", englishName: "Spanish" },
  { code: "ru", name: "俄语", englishName: "Russian" },
  { code: "pt", name: "葡萄牙语", englishName: "Portuguese" },
  { code: "it", name: "意大利语", englishName: "Italian" },
  { code: "ar", name: "阿拉伯语", englishName: "Arabic" },
  { code: "th", name: "泰语", englishName: "Thai" },
  { code: "vi", name: "越南语", englishName: "Vietnamese" },
];

export const targetLanguages = languages.filter((l) => l.code !== "auto");
