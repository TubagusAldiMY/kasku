import { createCrudRepository } from './crud';
import type { BudgetRow } from '../schema';

export const budgetsRepo = createCrudRepository<BudgetRow>('budgets');
