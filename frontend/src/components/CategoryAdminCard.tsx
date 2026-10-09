import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus, Trash2, X } from 'lucide-react'
import { SortableHead } from '@/components/SortableHead'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  createCategory,
  createSubCategory,
  deleteCategory,
  deleteSubCategory,
  fetchCategories,
  type Category,
} from '@/lib/api'
import { useNameCountSort } from '@/lib/use-name-count-sort'

const OTHER = 'Other'

export function CategoryAdminCard() {
  const queryClient = useQueryClient()
  const { data: categories = [], isLoading, error } = useQuery({
    queryKey: ['categories'],
    queryFn: fetchCategories,
  })
  const [name, setName] = useState('')

  const create = useMutation({
    mutationFn: () => createCategory(name),
    onSuccess: () => {
      setName('')
      queryClient.invalidateQueries({ queryKey: ['categories'] })
    },
  })
  const remove = useMutation({
    mutationFn: (categoryName: string) => deleteCategory(categoryName),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['categories'] }),
  })
  const mutationError = create.error ?? remove.error
  const { sorted, sort, toggle } = useNameCountSort(categories)

  return (
    <Card>
      <CardHeader>
        <CardTitle>Categories</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-muted-foreground text-sm">
          Each category splits into sub-categories (e.g. Equity → Stock, ETF). A category needs at
          least one sub-category before instruments can use it. Unused entries can be deleted.
        </p>

        <form
          className="flex flex-wrap gap-3"
          onSubmit={(e) => {
            e.preventDefault()
            create.mutate()
          }}
        >
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. Fixed Income"
            maxLength={50}
            className="max-w-sm"
            aria-label="New category name"
          />
          <Button type="submit" disabled={!name.trim() || create.isPending}>
            {create.isPending ? 'Creating…' : 'Create category'}
          </Button>
        </form>

        {mutationError && (
          <p className="text-destructive text-sm">
            {mutationError instanceof Error ? mutationError.message : 'Something went wrong.'}
          </p>
        )}
        {create.isSuccess && (
          <p className="text-sm">
            <span className="font-semibold">{create.data.name}</span> created. Add a sub-category
            to it before assigning instruments.
          </p>
        )}

        {isLoading && <p className="text-muted-foreground text-sm">Loading…</p>}
        {!!error && <p className="text-destructive text-sm">Failed to load categories.</p>}
        {categories.length > 0 && (
          <Table>
            <TableHeader>
              <TableRow>
                <SortableHead label="Category" sortKey="name" sort={sort} onToggle={toggle} />
                <SortableHead
                  label="Instruments"
                  sortKey="count"
                  sort={sort}
                  onToggle={toggle}
                  className="text-right"
                />
                <TableHead className="w-12" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {sorted.map((c) => {
                const deletable = c.instrumentCount === 0 && c.name !== OTHER
                return (
                  <TableRow key={c.name}>
                    <TableCell className="space-y-2 whitespace-normal">
                      <span className="font-medium">{c.name}</span>
                      <SubCategoryChips category={c} />
                    </TableCell>
                    <TableCell className="text-right align-top tabular-nums">{c.instrumentCount}</TableCell>
                    <TableCell className="text-right align-top">
                      {deletable && (
                        <Button
                          size="icon-sm"
                          variant="ghost"
                          aria-label={`Delete ${c.name}`}
                          disabled={remove.isPending}
                          onClick={() => {
                            if (window.confirm(`Delete the "${c.name}" category and its sub-categories?`)) {
                              remove.mutate(c.name)
                            }
                          }}
                        >
                          <Trash2 className="size-4" />
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  )
}

function SubCategoryChips({ category }: { category: Category }) {
  const queryClient = useQueryClient()
  const [name, setName] = useState('')
  const refresh = () => queryClient.invalidateQueries({ queryKey: ['categories'] })

  const add = useMutation({
    mutationFn: () => createSubCategory(category.name, name),
    onSuccess: () => {
      setName('')
      refresh()
    },
  })
  const remove = useMutation({
    mutationFn: (sub: string) => deleteSubCategory(category.name, sub),
    onSuccess: refresh,
  })
  const error = add.error ?? remove.error

  return (
    <div className="space-y-1">
      <div className="flex flex-wrap items-center gap-1.5">
        {category.subCategories.length === 0 && (
          <span className="text-muted-foreground text-xs">No sub-categories yet.</span>
        )}
        {category.subCategories.map((sub) => {
          const deletable = sub.instrumentCount === 0 && !(category.name === OTHER && sub.name === OTHER)
          return (
            <Badge key={sub.name} variant="secondary" className="gap-1 font-normal">
              {sub.name}
              <span className="text-muted-foreground">· {sub.instrumentCount}</span>
              {deletable && (
                <button
                  type="button"
                  aria-label={`Delete sub-category ${sub.name}`}
                  className="hover:text-destructive"
                  disabled={remove.isPending}
                  onClick={() => remove.mutate(sub.name)}
                >
                  <X className="size-3" />
                </button>
              )}
            </Badge>
          )
        })}
        <form
          className="flex items-center gap-1"
          onSubmit={(e) => {
            e.preventDefault()
            add.mutate()
          }}
        >
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Add sub-category"
            maxLength={50}
            className="h-6 w-32 text-xs"
            aria-label={`New sub-category of ${category.name}`}
          />
          <Button
            type="submit"
            size="icon-xs"
            variant="outline"
            aria-label="Add sub-category"
            disabled={!name.trim() || add.isPending}
          >
            <Plus className="size-3" />
          </Button>
        </form>
      </div>
      {error && (
        <p className="text-destructive text-xs">
          {error instanceof Error ? error.message : 'Something went wrong.'}
        </p>
      )}
    </div>
  )
}
