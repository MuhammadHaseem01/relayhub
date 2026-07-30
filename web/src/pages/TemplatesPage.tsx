import React, { useState, useEffect } from 'react';
import { Puzzle, Plus, Edit, Trash2, Code2, X } from 'lucide-react';
import { listTemplates, createTemplate, updateTemplate, deleteTemplate, type TemplateRecord } from '../api/client';
import { ConfirmDialog } from '../components/ConfirmDialog';

interface TemplatesPageProps {
  onShowToast: (message: string, type: 'success' | 'error') => void;
}

export const TemplatesPage: React.FC<TemplatesPageProps> = ({ onShowToast }) => {
  const [templates, setTemplates] = useState<TemplateRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Modal / Form state
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isEdit, setIsEdit] = useState(false);
  const [name, setName] = useState('');
  const [body, setBody] = useState('');
  const [submitting, setSubmitting] = useState(false);

  // Delete Confirm Dialog state
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [deleting, setDeleting] = useState(false);

  const fetchTemplates = async (showLoading = false) => {
    if (showLoading) setLoading(true);
    setError(null);
    const res = await listTemplates();
    setLoading(false);

    if (res.success && res.data) {
      setTemplates(res.data.templates || []);
    } else {
      setError(res.error || 'Failed to load templates');
    }
  };

  useEffect(() => {
    fetchTemplates(true);
  }, []);

  // Extract Handlebars variables client-side for live UX preview
  const parseVariables = (text: string): string[] => {
    const matches = text.match(/\{\{\s*([a-zA-Z0-9_]+)\s*\}\}/g);
    if (!matches) return [];
    const vars = matches.map(m => m.replace(/[\{\}\s]/g, ''));
    return Array.from(new Set(vars));
  };

  const detectedVars = parseVariables(body);

  const handleOpenCreate = () => {
    setIsEdit(false);
    setName('');
    setBody('');
    setIsModalOpen(true);
  };

  const handleOpenEdit = (tmpl: TemplateRecord) => {
    setIsEdit(true);
    setName(tmpl.name);
    setBody(tmpl.body);
    setIsModalOpen(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !body.trim()) return;

    setSubmitting(true);
    const res = isEdit
      ? await updateTemplate(name.trim(), body.trim())
      : await createTemplate(name.trim(), body.trim());
    setSubmitting(false);

    if (res.success) {
      onShowToast(`Template "${name}" ${isEdit ? 'updated' : 'created'} successfully`, 'success');
      setIsModalOpen(false);
      fetchTemplates(false);
    } else {
      onShowToast(res.error || 'Failed to save template', 'error');
    }
  };

  const handleDeleteConfirm = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    const res = await deleteTemplate(deleteTarget);
    setDeleting(false);

    if (res.success) {
      onShowToast(`Template "${deleteTarget}" deleted`, 'success');
      setDeleteTarget(null);
      fetchTemplates(false);
    } else {
      onShowToast(res.error || 'Failed to delete template', 'error');
    }
  };

  return (
    <div className="card">
      {/* Top Header & Actions */}
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        marginBottom: '20px',
        flexWrap: 'wrap',
        gap: '12px'
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <Puzzle size={20} style={{ color: 'var(--accent)' }} />
          <h3 style={{ fontSize: '16px', margin: 0 }}>Template Management</h3>
          <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
            ({templates.length} templates)
          </span>
        </div>

        <button className="btn btn-primary" onClick={handleOpenCreate}>
          <Plus size={16} />
          <span>New Template</span>
        </button>
      </div>

      {error && (
        <div style={{
          backgroundColor: 'rgba(248, 81, 73, 0.15)',
          border: '1px solid var(--danger)',
          color: 'var(--danger)',
          padding: '10px',
          borderRadius: 'var(--radius-md)',
          fontSize: '13px',
          marginBottom: '16px'
        }}>
          {error}
        </div>
      )}

      {loading && templates.length === 0 ? (
        <div style={{ padding: '40px', textAlign: 'center', color: 'var(--text-secondary)' }}>
          Loading templates...
        </div>
      ) : templates.length === 0 ? (
        <div className="empty-state">
          <Puzzle size={40} className="empty-state-icon" />
          <p className="empty-state-text">No notification templates found. Create reusable templates with dynamic {'{{variables}}'}!</p>
          <button className="btn btn-primary btn-sm" onClick={handleOpenCreate}>
            <Plus size={14} />
            <span>Create Template</span>
          </button>
        </div>
      ) : (
        <div style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px', textAlign: 'left' }}>
            <thead>
              <tr style={{ borderBottom: '1px solid var(--border)', color: 'var(--text-secondary)' }}>
                <th style={{ padding: '10px 12px' }}>Name</th>
                <th style={{ padding: '10px 12px' }}>Body Preview</th>
                <th style={{ padding: '10px 12px' }}>Created At</th>
                <th style={{ padding: '10px 12px' }}>Updated At</th>
                <th style={{ padding: '10px 12px', textAlign: 'right' }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {templates.map((tmpl) => (
                <tr key={tmpl.id} style={{ borderBottom: '1px solid var(--border-strong)' }}>
                  <td style={{ padding: '10px 12px', fontFamily: 'var(--font-mono)', color: 'var(--accent)', fontWeight: 600 }}>
                    {tmpl.name}
                  </td>
                  <td style={{
                    padding: '10px 12px',
                    maxWidth: '300px',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                    color: 'var(--text-secondary)',
                    fontFamily: 'var(--font-mono)',
                    fontSize: '12px'
                  }}>
                    {tmpl.body}
                  </td>
                  <td style={{ padding: '10px 12px', color: 'var(--text-faint)', fontSize: '12px' }}>
                    {new Date(tmpl.created_at).toLocaleDateString()}
                  </td>
                  <td style={{ padding: '10px 12px', color: 'var(--text-faint)', fontSize: '12px' }}>
                    {new Date(tmpl.updated_at).toLocaleDateString()}
                  </td>
                  <td style={{ padding: '10px 12px', textAlign: 'right' }}>
                    <div style={{ display: 'inline-flex', gap: '8px' }}>
                      <button className="btn btn-sm btn-ghost" onClick={() => handleOpenEdit(tmpl)} title="Edit Template">
                        <Edit size={14} />
                      </button>
                      <button className="btn btn-sm btn-danger" onClick={() => setDeleteTarget(tmpl.name)} title="Delete Template">
                        <Trash2 size={14} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Create / Edit Template Modal */}
      {isModalOpen && (
        <div style={{
          position: 'fixed',
          inset: 0,
          backgroundColor: 'rgba(13, 17, 23, 0.85)',
          backdropFilter: 'blur(4px)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          zIndex: 1000,
          padding: '20px'
        }}>
          <div className="card" style={{ width: '100%', maxWidth: '520px', boxShadow: '0 20px 40px rgba(0,0,0,0.6)' }}>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <Code2 size={18} style={{ color: 'var(--accent)' }} />
                <h3 style={{ fontSize: '16px', margin: 0 }}>
                  {isEdit ? `Edit Template "${name}"` : 'Create New Template'}
                </h3>
              </div>
              <button className="btn btn-sm btn-ghost" onClick={() => setIsModalOpen(false)}>
                <X size={16} />
              </button>
            </div>

            <form onSubmit={handleSubmit}>
              <div className="form-group">
                <label className="form-label">Template Name</label>
                <input
                  type="text"
                  className="form-input font-mono"
                  placeholder="e.g. order_shipped"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  disabled={isEdit}
                  required
                />
                <span className="form-hint">Alphanumeric and underscores only, max 64 characters.</span>
              </div>

              <div className="form-group">
                <label className="form-label">Template Body</label>
                <textarea
                  className="form-textarea font-mono"
                  rows={6}
                  placeholder="Hi {{name}}, your order {{order_id}} is ready!"
                  value={body}
                  onChange={(e) => setBody(e.target.value)}
                  required
                />
                <span className="form-hint">{body.length} / 4000 characters</span>
              </div>

              {/* Detected Variables Chips */}
              <div style={{ marginBottom: '20px', backgroundColor: 'var(--bg)', padding: '12px', borderRadius: 'var(--radius-md)', border: '1px solid var(--border)' }}>
                <span style={{ fontSize: '12px', color: 'var(--text-secondary)', fontWeight: 600, display: 'block', marginBottom: '6px' }}>
                  Detected Handlebars Variables:
                </span>
                {detectedVars.length === 0 ? (
                  <span style={{ fontSize: '12px', color: 'var(--text-faint)' }}>None (use {'{{variable_name}}'} syntax)</span>
                ) : (
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
                    {detectedVars.map(v => (
                      <span key={v} style={{
                        fontSize: '11px',
                        fontFamily: 'var(--font-mono)',
                        padding: '2px 8px',
                        borderRadius: '12px',
                        backgroundColor: 'var(--accent-bg)',
                        color: 'var(--accent)',
                        border: '1px solid rgba(88, 166, 255, 0.3)'
                      }}>
                        {`{{${v}}}`}
                      </span>
                    ))}
                  </div>
                )}
              </div>

              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px' }}>
                <button type="button" className="btn btn-ghost" onClick={() => setIsModalOpen(false)}>
                  Cancel
                </button>
                <button type="submit" className="btn btn-primary" disabled={submitting}>
                  {submitting ? 'Saving...' : isEdit ? 'Update Template' : 'Create Template'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Delete Confirmation Dialog */}
      <ConfirmDialog
        isOpen={Boolean(deleteTarget)}
        title="Delete Template"
        message={`Are you sure you want to delete template "${deleteTarget}"? Requests referencing this template will fail.`}
        confirmText="Delete Template"
        isDanger={true}
        loading={deleting}
        onConfirm={handleDeleteConfirm}
        onCancel={() => setDeleteTarget(null)}
      />
    </div>
  );
};
