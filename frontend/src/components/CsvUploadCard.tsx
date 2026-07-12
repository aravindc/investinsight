import { useEffect, useRef, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { fetchAccounts, uploadCsv, type Account, type CsvUploadResult } from '@/lib/api'

export function CsvUploadCard() {
  const queryClient = useQueryClient()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [result, setResult] = useState<CsvUploadResult | null>(null)
  const [accounts, setAccounts] = useState<Account[]>([])
  const [accountId, setAccountId] = useState<string>('')

  useEffect(() => {
    fetchAccounts().then((fetched) => {
      setAccounts(fetched)
      if (fetched.length > 0) setAccountId(fetched[0].id)
    })
  }, [])

  const mutation = useMutation({
    mutationFn: (file: File) => uploadCsv(file, true, accountId),
    onSuccess: (data) => {
      setResult(data)
      queryClient.invalidateQueries({ queryKey: ['transactions'] })
      queryClient.invalidateQueries({ queryKey: ['holdings-summary'] })
      queryClient.invalidateQueries({ queryKey: ['dividends-summary'] })
      setSelectedFile(null)
      if (fileInputRef.current) fileInputRef.current.value = ''
    },
  })

  const handleUpload = () => {
    if (!selectedFile || !accountId) return
    setResult(null)
    mutation.mutate(selectedFile)
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Import transactions</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex gap-3">
          {accounts.length > 1 && (
            <Select value={accountId} onValueChange={setAccountId}>
              <SelectTrigger className="w-40">
                <SelectValue placeholder="Account" />
              </SelectTrigger>
              <SelectContent>
                {accounts.map((a) => (
                  <SelectItem key={a.id} value={a.id}>
                    {a.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
          <Input
            ref={fileInputRef}
            type="file"
            accept=".csv"
            onChange={(e) => setSelectedFile(e.target.files?.[0] ?? null)}
            className="max-w-sm"
          />
          <Button onClick={handleUpload} disabled={!selectedFile || !accountId || mutation.isPending}>
            {mutation.isPending ? 'Uploading…' : 'Upload CSV'}
          </Button>
        </div>

        {mutation.isError && (
          <p className="text-destructive text-sm">
            {mutation.error instanceof Error ? mutation.error.message : 'Upload failed.'}
          </p>
        )}

        {result && (
          <div className="text-sm">
            <p>
              <span className="font-semibold">{result.inserted}</span> new transaction
              {result.inserted === 1 ? '' : 's'} imported,{' '}
              <span className="font-semibold">{result.duplicate}</span> duplicate
              {result.duplicate === 1 ? '' : 's'} skipped.
            </p>
            {result.errors.length > 0 && (
              <div className="text-destructive mt-2">
                <p>{result.errors.length} row(s) could not be parsed:</p>
                <ul className="list-disc pl-5">
                  {result.errors.slice(0, 5).map((err, i) => (
                    <li key={i}>{err}</li>
                  ))}
                </ul>
                {result.errors.length > 5 && (
                  <p>…and {result.errors.length - 5} more.</p>
                )}
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
