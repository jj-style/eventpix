export type Result<T, E = string> = { data: T; ok: true } | { error: E; ok: false };

export function wrap<T, E = string>(data: T): Result<T, E> {
	return { data, ok: true };
}

export function unwrap<T, E = string>(result: Result<T, E>): T {
	if (result.ok) {
		return result.data;
	} else {
		throw new Error(`${result.error}`);
	}
}
