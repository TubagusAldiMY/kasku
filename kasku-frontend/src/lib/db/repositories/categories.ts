import { createCrudRepository } from './crud';
import type { CategoryRow } from '../schema';

export const categoriesRepo = createCrudRepository<CategoryRow>('categories');
