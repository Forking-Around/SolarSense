create extension if not exists vector with schema extensions;

create table public.assessments (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references auth.users(id) on delete cascade,
  status text not null default 'draft' check (status in ('draft','complete')),
  input jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(), updated_at timestamptz not null default now()
);
create index assessments_user_created_idx on public.assessments (user_id, created_at desc);

create table public.locations (
  id uuid primary key default gen_random_uuid(), assessment_id uuid not null references public.assessments(id) on delete cascade,
  user_id uuid not null references auth.users(id) on delete cascade,
  display_name text not null, country_code text not null, state text, postal_code text, utility text,
  latitude double precision not null check (latitude between -90 and 90), longitude double precision not null check (longitude between -180 and 180),
  source text not null, confirmed_at timestamptz
);
create index locations_assessment_idx on public.locations (assessment_id);

create table public.bill_extractions (
  id uuid primary key default gen_random_uuid(), assessment_id uuid not null references public.assessments(id) on delete cascade,
  user_id uuid not null references auth.users(id) on delete cascade,
  storage_path text, extracted jsonb not null, confidence jsonb not null default '{}'::jsonb,
  provider text not null, confirmed_at timestamptz, created_at timestamptz not null default now()
);

create table public.roof_observations (
  id uuid primary key default gen_random_uuid(), assessment_id uuid not null references public.assessments(id) on delete cascade,
  user_id uuid not null references auth.users(id) on delete cascade,
  storage_path text, observation jsonb not null, provider text, confirmed_at timestamptz, created_at timestamptz not null default now()
);

create table public.reports (
  id uuid primary key default gen_random_uuid(), assessment_id uuid not null references public.assessments(id) on delete cascade,
  user_id uuid not null references auth.users(id) on delete cascade,
  report jsonb not null, engine_version text not null, solar_data_version text not null, policy_version text not null,
  created_at timestamptz not null default now()
);
create index reports_user_created_idx on public.reports (user_id, created_at desc);

create table public.policy_packs (
  id uuid primary key default gen_random_uuid(), jurisdiction text not null, utility text, version text not null,
  effective_from date not null, effective_to date, rules jsonb not null,
  source_url text not null, review_status text not null check (review_status in ('candidate','reviewed','superseded')),
  reviewed_at timestamptz, created_at timestamptz not null default now(), unique(jurisdiction, utility, version)
);
create index policy_active_lookup_idx on public.policy_packs (jurisdiction, utility, effective_from desc) where review_status='reviewed';

create table public.knowledge_documents (
  id uuid primary key default gen_random_uuid(), title text not null, content text not null, language text not null default 'en',
  country_code text, state text, utility text, topic text not null, source_url text not null,
  effective_from date, effective_to date, review_status text not null check (review_status in ('candidate','reviewed','superseded')),
  search_vector tsvector generated always as (setweight(to_tsvector('simple', coalesce(title,'')), 'A') || setweight(to_tsvector('simple', coalesce(content,'')), 'B')) stored,
  embedding extensions.vector(768), reviewed_at timestamptz, created_at timestamptz not null default now()
);
create index knowledge_search_idx on public.knowledge_documents using gin(search_vector);
create index knowledge_filters_idx on public.knowledge_documents (country_code,state,utility,topic,review_status);

create table public.solar_data_cache (
  cache_key text primary key, latitude double precision not null, longitude double precision not null,
  source text not null, source_version text not null, values jsonb not null, retrieved_at timestamptz not null, expires_at timestamptz not null
);
create index solar_cache_expiry_idx on public.solar_data_cache (expires_at);

create table public.provider_audit (
  id bigint generated always as identity primary key, user_id uuid references auth.users(id) on delete set null,
  purpose text not null, provider text not null, model text not null, outcome text not null,
  latency_ms integer, created_at timestamptz not null default now()
);

alter table public.assessments enable row level security;
alter table public.locations enable row level security;
alter table public.bill_extractions enable row level security;
alter table public.roof_observations enable row level security;
alter table public.reports enable row level security;
alter table public.policy_packs enable row level security;
alter table public.knowledge_documents enable row level security;
alter table public.solar_data_cache enable row level security;
alter table public.provider_audit enable row level security;

create policy policy_reviewed_read on public.policy_packs for select to anon,authenticated using (review_status='reviewed' and effective_from<=current_date and (effective_to is null or effective_to>=current_date));
create policy knowledge_reviewed_read on public.knowledge_documents for select to anon,authenticated using (review_status='reviewed' and (effective_from is null or effective_from<=current_date) and (effective_to is null or effective_to>=current_date));

create policy assessments_owner_select on public.assessments for select to authenticated using ((select auth.uid()) = user_id);
create policy assessments_owner_insert on public.assessments for insert to authenticated with check ((select auth.uid()) = user_id);
create policy assessments_owner_update on public.assessments for update to authenticated using ((select auth.uid()) = user_id) with check ((select auth.uid()) = user_id);
create policy assessments_owner_delete on public.assessments for delete to authenticated using ((select auth.uid()) = user_id);

create policy locations_owner_all on public.locations for all to authenticated using ((select auth.uid()) = user_id) with check ((select auth.uid()) = user_id);
create policy bills_owner_all on public.bill_extractions for all to authenticated using ((select auth.uid()) = user_id) with check ((select auth.uid()) = user_id);
create policy roofs_owner_all on public.roof_observations for all to authenticated using ((select auth.uid()) = user_id) with check ((select auth.uid()) = user_id);
create policy reports_owner_all on public.reports for all to authenticated using ((select auth.uid()) = user_id) with check ((select auth.uid()) = user_id);

insert into storage.buckets (id,name,public,file_size_limit,allowed_mime_types)
values ('private-assessment-uploads','private-assessment-uploads',false,10485760,array['image/jpeg','image/png','application/pdf'])
on conflict (id) do update set public=false,file_size_limit=excluded.file_size_limit,allowed_mime_types=excluded.allowed_mime_types;

create policy upload_owner_select on storage.objects for select to authenticated using (bucket_id='private-assessment-uploads' and (storage.foldername(name))[1]=(select auth.uid())::text);
create policy upload_owner_insert on storage.objects for insert to authenticated with check (bucket_id='private-assessment-uploads' and (storage.foldername(name))[1]=(select auth.uid())::text);
create policy upload_owner_update on storage.objects for update to authenticated using (bucket_id='private-assessment-uploads' and (storage.foldername(name))[1]=(select auth.uid())::text) with check (bucket_id='private-assessment-uploads' and (storage.foldername(name))[1]=(select auth.uid())::text);
create policy upload_owner_delete on storage.objects for delete to authenticated using (bucket_id='private-assessment-uploads' and (storage.foldername(name))[1]=(select auth.uid())::text);

grant select,insert,update,delete on public.assessments,public.locations,public.bill_extractions,public.roof_observations,public.reports to authenticated;
grant select on public.policy_packs,public.knowledge_documents to anon,authenticated;
