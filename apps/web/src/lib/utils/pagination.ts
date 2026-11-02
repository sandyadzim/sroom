export function getPagination(total: number, requestedPage: number, pageSize: number) {
	const totalPages = Math.max(1, Math.ceil(total / pageSize));
	const page = Math.max(1, Math.min(totalPages, Math.trunc(requestedPage) || 1));
	const offset = (page - 1) * pageSize;
	return {
		page,
		totalPages,
		offset,
		start: total === 0 ? 0 : offset + 1,
		end: Math.min(offset + pageSize, total)
	};
}
