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
};
