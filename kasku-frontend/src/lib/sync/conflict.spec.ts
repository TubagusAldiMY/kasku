import { describe, expect, it } from 'vitest';
import { applyServerWins } from './conflict';
import type { AccountRow } from '$lib/db';

function makeAccount(overrides: Partial<AccountRow> = {}): AccountRow {
	return {
		id: 'acc-1',
		name: 'BCA',
		account_type: 'BANK',
		balance: 0,
		currency: 'IDR',
		updated_at: '2026-05-17T00:00:00.000Z',
		...overrides
	};
}

describe('applyServerWins', () => {
	it('accepts server when no local copy exists', () => {
		const server = makeAccount();
		const res = applyServerWins<AccountRow>(undefined, server);
		expect(res.decision).toBe('apply_server');
		expect(res.winner).toBe(server);
		expect(res.loserHadLocalChanges).toBe(false);
	});

	it('server tombstone wins even if local is newer + dirty', () => {
		const local = makeAccount({ updated_at: '2026-05-17T12:00:00.000Z', _local_dirty: true });
		const server = makeAccount({ updated_at: '2026-05-17T10:00:00.000Z', _deleted: true });
		const res = applyServerWins(local, server);
		expect(res.decision).toBe('apply_server');
		expect(res.loserHadLocalChanges).toBe(true);
	});

	it('server wins when server timestamp is newer', () => {
		const local = makeAccount({ updated_at: '2026-05-17T10:00:00.000Z' });
		const server = makeAccount({ updated_at: '2026-05-17T11:00:00.000Z' });
		const res = applyServerWins(local, server);
		expect(res.decision).toBe('apply_server');
	});

	it('server wins on tie', () => {
		const local = makeAccount({ updated_at: '2026-05-17T10:00:00.000Z' });
		const server = makeAccount({ updated_at: '2026-05-17T10:00:00.000Z' });
		const res = applyServerWins(local, server);
		expect(res.decision).toBe('apply_server');
	});

	it('keeps local when local is newer than server', () => {
		const local = makeAccount({
			updated_at: '2026-05-17T12:00:00.000Z',
			_local_dirty: true
		});
		const server = makeAccount({ updated_at: '2026-05-17T10:00:00.000Z' });
		const res = applyServerWins(local, server);
		expect(res.decision).toBe('keep_local');
		expect(res.winner).toBe(local);
	});

	it('flags conflict when server wins but local was dirty', () => {
		const local = makeAccount({
			updated_at: '2026-05-17T10:00:00.000Z',
			_local_dirty: true
		});
		const server = makeAccount({ updated_at: '2026-05-17T11:00:00.000Z' });
		const res = applyServerWins(local, server);
		expect(res.decision).toBe('apply_server');
		expect(res.loserHadLocalChanges).toBe(true);
	});

	it('falls back to keep_local if server timestamp is invalid', () => {
		const local = makeAccount();
		const server = makeAccount({ updated_at: 'not-a-date' });
		const res = applyServerWins(local, server);
		expect(res.decision).toBe('keep_local');
	});
});
