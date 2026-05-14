import { type ColumnDef, flexRender, getCoreRowModel, useReactTable } from "@tanstack/react-table";

type DataTableProps<T> = {
    data: T[];
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    columns: ColumnDef<T, any>[];
    onRowClick?: (row: T) => void;
    selectedRowId?: string;
    getRowId?: (row: T) => string;
    getRowClassName?: (row: T) => string | undefined;
    className?: string;
    wrapClassName?: string;
};

export function DataTable<T>({
    data,
    columns,
    onRowClick,
    selectedRowId,
    getRowId,
    getRowClassName,
    className,
    wrapClassName,
}: DataTableProps<T>) {
    // TanStack Table returns mutable table helpers; React Compiler intentionally skips memoizing here.
    // eslint-disable-next-line react-hooks/incompatible-library
    const table = useReactTable({
        data,
        columns,
        getCoreRowModel: getCoreRowModel(),
        getRowId,
    });

    return (
        <div className={`data-table-wrap ${wrapClassName ?? ""}`}>
            <table className={`data-table ${className ?? ""}`}>
                <thead>
                    {table.getHeaderGroups().map((headerGroup) => (
                        <tr key={headerGroup.id}>
                            {headerGroup.headers.map((header) => (
                                <th key={header.id}>
                                    {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                                </th>
                            ))}
                        </tr>
                    ))}
                </thead>
                <tbody>
                    {table.getRowModel().rows.map((row) => {
                        const isSelected = selectedRowId != null && row.id === selectedRowId;
                        const extraClass = getRowClassName ? (getRowClassName(row.original) ?? "") : "";
                        const baseClass = isSelected ? "data-table-row-selected" : onRowClick ? "clickable-row" : "";
                        return (
                            <tr
                                key={row.id}
                                className={[baseClass, extraClass].filter(Boolean).join(" ") || undefined}
                                onClick={onRowClick ? () => onRowClick(row.original) : undefined}
                            >
                                {row.getVisibleCells().map((cell) => (
                                    <td key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</td>
                                ))}
                            </tr>
                        );
                    })}
                </tbody>
            </table>
        </div>
    );
}
