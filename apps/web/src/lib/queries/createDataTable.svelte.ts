import { afterNavigate, goto } from '$app/navigation';
import { page as currentPage } from '$app/state';
import { createQuery } from '@tanstack/svelte-query';
import { isAxiosError } from 'axios';
import { onDestroy, untrack } from 'svelte';
import { SvelteSet, SvelteURL, SvelteURLSearchParams } from 'svelte/reactivity';
import { fetchTablePage } from '#lib/api/data-table.ts';
import type { DataTableConfig, TableFilter, TableParams } from '#lib/types/data-table.ts';
import type { ApiError } from '#lib/types/identity.ts';
import { getPagination } from '#lib/utils/pagination.ts';

function allowedFilters(
	search: Pick<URLSearchParams, 'get'>,
	definitions: readonly TableFilter[] = []
) {
	const allowed = new SvelteURLSearchParams();
	for (const filter of definitions) {
		if (filter.name === 'page' || filter.name === 'limit') continue;
		const value = search.get(filter.name) ?? '';
		if (filter.type === 'search') {
			const term = value.trim();
			if (term) allowed.set(filter.name, term);
		} else if (value && filter.options.some((option) => option.value === value)) {
			allowed.set(filter.name, value);
		}
	}
	return allowed.toString();
}

function readPositiveInteger(value: string | null) {
	const number = value && /^\d+$/.test(value) ? Number(value) : 0;
	return Number.isSafeInteger(number) && number > 0 ? number : undefined;
}

export function createDataTable<Row>(getConfig: () => DataTableConfig<Row>) {
	let searchDrafts = $state<Record<string, string>>({});
	const searchTimers: Record<string, ReturnType<typeof setTimeout>> = Object.create(null);
	const config = $derived(getConfig());
	const url = $derived(currentPage.shallow?.url ?? currentPage.url);
	const filterQuery = $derived(allowedFilters(url.searchParams, config.filters));
	const filters = $derived(Object.fromEntries(new SvelteURLSearchParams(filterQuery)));
	const pageSizes = $derived(
		[...new SvelteSet([...(config.pageSizes ?? [10, 20, 50, 100]), config.pageSize])].sort(
			(a, b) => a - b
		)
	);
	const page = $derived(readPositiveInteger(url.searchParams.get('page')) ?? 1);
	const requestedLimit = $derived(readPositiveInteger(url.searchParams.get('limit')));
	const pageSize = $derived(
		requestedLimit !== undefined && pageSizes.includes(requestedLimit)
			? requestedLimit
			: config.pageSize
	);
	const params: TableParams = $derived({ page, limit: pageSize, filters });

	afterNavigate(({ type, shallow }) => {
		if (type === 'goto' && shallow) return;
		for (const name of Object.keys(searchTimers)) cancelSearch(name);
		searchDrafts = {};
	});

	const query = createQuery(() => {
		const endpoint = config.endpoint;
		const requestParams = params;
		return {
			queryKey: [endpoint, requestParams],
			queryFn: ({ signal }) => fetchTablePage<Row>(endpoint, requestParams, signal),
			refetchOnWindowFocus: false
		};
	});

	const pagination = $derived(
		getPagination(query.data?.page.total ?? 0, page, query.data?.page.limit ?? pageSize)
	);

	$effect(() => {
		const validPage = query.isSuccess && !query.isFetching ? pagination.page : page;
		const search = buildSearch(filterQuery, validPage, pageSize);
		if (url.search.slice(1) !== search) untrack(() => replaceSearch(search));
	});

	const failure = $derived.by(() => {
		if (query.isError) {
			const error = query.error;
			const response = isAxiosError<ApiError>(error) ? error.response : undefined;
			return {
				status: response?.status,
				message: response?.data?.error?.message ?? 'Could not load results. Please try again.'
			};
		}
	});

	function setFilter(name: string, value: string) {
		const next = new SvelteURLSearchParams(filterQuery);
		if (value) next.set(name, value);
		else next.delete(name);
		const updated = allowedFilters(next, config.filters);
		if (updated !== filterQuery) replaceSearch(buildSearch(updated, 1, pageSize));
	}

	function buildSearch(filterSearch: string, nextPage: number, limit: number) {
		const search = new SvelteURLSearchParams(config.urlContext);
		for (const [key, value] of new SvelteURLSearchParams(filterSearch)) search.set(key, value);
		if (nextPage > 1) search.set('page', String(nextPage));
		if (limit !== config.pageSize) search.set('limit', String(limit));
		return search.toString();
	}

	function replaceSearch(search: string) {
		const next = new SvelteURL(url.href);
		next.search = search;
		// The current page URL already includes the resolved route and base path.
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		void goto(next, { shallow: true, replace: true, state: currentPage.state });
	}

	function cancelSearch(name: string) {
		clearTimeout(searchTimers[name]);
		delete searchTimers[name];
	}

	function updateSearch(name: string, value: string, composing = false) {
		searchDrafts[name] = value;
		cancelSearch(name);
		if (composing) return;

		searchTimers[name] = setTimeout(() => {
			delete searchTimers[name];
			setFilter(name, value);
		}, 300);
	}

	function clearSearch(name: string) {
		cancelSearch(name);
		searchDrafts[name] = '';
		setFilter(name, '');
	}

	onDestroy(() => Object.values(searchTimers).forEach(clearTimeout));

	function changePageSize(size: number) {
		if (size !== pageSize && pageSizes.includes(size)) {
			replaceSearch(buildSearch(filterQuery, 1, size));
		}
	}

	function changePage(next: number) {
		if (query.isSuccess && next !== page && next >= 1 && next <= pagination.totalPages) {
			replaceSearch(buildSearch(filterQuery, next, pageSize));
		}
	}

	return {
		query,
		get page() {
			return page;
		},
		get pageSize() {
			return pageSize;
		},
		get pageSizes() {
			return pageSizes;
		},
		get pagination() {
			return pagination;
		},
		get filters() {
			return filters;
		},
		searchValue(name: string) {
			return searchDrafts[name] ?? filters[name] ?? '';
		},
		get failure() {
			return failure;
		},
		updateSearch,
		clearSearch,
		changePageSize,
		setFilter,
		changePage,
		refetch: () => void query.refetch()
	};
}
