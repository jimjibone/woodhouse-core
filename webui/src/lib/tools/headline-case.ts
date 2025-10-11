export function toHeadlineCase(input: string): string {
	const minorWords = new Set([
		"a", "an", "the", "and", "but", "or", "for", "nor",
		"on", "in", "at", "to", "from", "by", "with", "of", "over"
	]);

	const words = input.toLowerCase().split(/\s+/);

	return words
		.map((word, index) => {
		if (
			index === 0 ||
			index === words.length - 1 ||
			!minorWords.has(word)
		) {
			return word[0].toUpperCase() + word.slice(1);
		} else {
			return word;
		}
		})
		.join(" ");
}

export function toSentenceCase(str: string): string {
	if (!str) return '';
	return str.replace(/^\s*([a-z])/, (_, c) => c.toUpperCase());
}

export function toSentenceCaseLocalized(str: string, locale: string = navigator.language): string {
	if (!str) return '';

	// Trim only the start (don't alter internal spacing)
	const trimmed = str.trimStart();

	// Find the first letter and capitalize it using locale rules
	return trimmed.replace(
		/^(\p{L})/u, // match first Unicode letter
		(c) => c.toLocaleUpperCase(locale)
	);
}
